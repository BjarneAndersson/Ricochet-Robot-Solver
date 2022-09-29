package solver

import (
	"../config"
	"../helper"
	"../output"
	"../priorityQueue"
	"../tracker"
	"../types"
)

func reconstructPath(cameFrom map[types.BoardState]types.BoardState, currentBoardState types.BoardState) (path []types.BoardState, err error) {
	path = append(path, currentBoardState)

	for {
		if _, ok := cameFrom[currentBoardState]; ok {
			currentBoardState = cameFrom[currentBoardState]
			path = append(path, currentBoardState)
		} else {
			break
		}
	}

	// reverse the path
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}

func Solver(board *types.Board, initBoardState types.BoardState, conf config.Config) (tracker.TrackingDataSolver, []types.BoardState, error) {
	trackingData := tracker.TrackingDataSolver{}

	openSet := make(priorityQueue.PriorityQueue, 1)
	closedSet := make([]types.BoardState, 0)

	openSet[0] = priorityQueue.Item{
		Value:    initBoardState,
		Priority: 0,
	}
	trackingData.InitializedBoardStates += 1

	cameFrom := make(map[types.BoardState]types.BoardState)

	gScore := make(map[types.BoardState]uint8) // g score - distance from start
	gScore[initBoardState] = 0

	// g score + h score -> value to evaluate the priority of the current board state
	fScore := make(map[types.BoardState]uint8)
	fScore[initBoardState] = calcFScore(board, initBoardState, gScore[initBoardState])

	for openSet.Len() > 0 {
		currentBoardState := priorityQueue.Pop(&openSet).Value

		if conf.Modes[conf.Mode]["output"].BoardStates == true {
			err := output.BoardState(currentBoardState, board.RobotColors)
			if err != nil {
				return trackingData, []types.BoardState{}, nil
			}
		}

		for indexRobot, robot := range helper.SeparateRobots(currentBoardState) {
			robotPosition := helper.ConvBytePositionToPosition(robot)
			node := board.Board[robotPosition.Row][robotPosition.Column]
			nodePosition := types.Position{Column: robotPosition.Column, Row: robotPosition.Row}

			for _, direction := range []string{"top", "bottom", "left", "right"} {
				cNode := node
				cNodePosition := nodePosition

				cNode, cNodePosition = calculateStoppingPosition(board, currentBoardState, cNode, cNodePosition, direction)

				if cNode != node { // robot can be moved into direction
					// move robot
					newRobots := moveRobot(helper.SeparateRobots(currentBoardState), uint8(indexRobot), cNodePosition)

					// create a new board state
					newBoardState := createNewBoardState(newRobots)

					// check if the new board state is already in the queue
					if isBoardStateInOpenSet(openSet, newBoardState) || isBoardStateInClosedSet(&closedSet, newBoardState) {
						continue
					}

					trackingData.InitializedBoardStates += 1

					// check if the new board state is the target
					// break -> reconstruct path
					if indexRobot == 0 { // if active robot was moved - only action to get to the target
						if isRobotOnTarget(&newBoardState, board.Target) {
							// add board state to cameFrom
							cameFrom[newBoardState] = currentBoardState
							path, err := reconstructPath(cameFrom, newBoardState)
							return trackingData, path, err
						}
					}

					// calc fScore for the new board state
					gScore[newBoardState] = gScore[currentBoardState] + 1
					currentFScore := calcFScore(board, newBoardState, gScore[newBoardState])

					// add board state to cameFrom
					cameFrom[newBoardState] = currentBoardState

					// add the new board state to the queue
					openSet.Push(
						priorityQueue.Item{
							Value:    newBoardState,
							Priority: int(currentFScore),
						})
				}

			}
		}
		trackingData.EvaluatedBoardStates += 1
		closedSet = append(closedSet, currentBoardState)
	}
	return trackingData, []types.BoardState{}, nil
}

func calcFScore(board *types.Board, boardState types.BoardState, gScore uint8) (fScore uint8) {
	fScore = gScore + calcHScore(board, boardState)
	return fScore
}

func calcHScore(board *types.Board, boardState types.BoardState) (hScore uint8) {
	activeRobotPosition := helper.ConvBytePositionToPosition(uint8((boardState & (255 << 24)) >> 24))

	node := board.Board[activeRobotPosition.Row][activeRobotPosition.Column]

	hScore = helper.GetMoveCount(node)
	return hScore
}

func createNewBoardState(robots [4]byte) types.BoardState {
	return types.BoardState(uint32(robots[0])<<24 | uint32(robots[1])<<16 | uint32(robots[2])<<8 | uint32(robots[3])<<0)
}

func calculateStoppingPosition(board *types.Board, boardState types.BoardState, startNode byte, startNodePosition types.Position, direction string) (cNode byte, cNodePosition types.Position) {
	cNode = startNode
	cNodePosition = startNodePosition
stopPositionLoop:
	for helper.HasNeighbor(cNode, direction) {

		switch direction {
		case "left":
			if checkRobotOnNode(boardState, types.Position{Column: cNodePosition.Column - 1, Row: cNodePosition.Row}) {
				break stopPositionLoop
			} else {
				cNodePosition = types.Position{Column: cNodePosition.Column - 1, Row: cNodePosition.Row}
				cNode = board.Board[cNodePosition.Row][cNodePosition.Column]
			}
		case "right":
			if checkRobotOnNode(boardState, types.Position{Column: cNodePosition.Column + 1, Row: cNodePosition.Row}) {
				break stopPositionLoop
			} else {
				cNodePosition = types.Position{Column: cNodePosition.Column + 1, Row: cNodePosition.Row}
				cNode = board.Board[cNodePosition.Row][cNodePosition.Column]
			}
		case "top":
			if checkRobotOnNode(boardState, types.Position{Column: cNodePosition.Column, Row: cNodePosition.Row - 1}) {
				break stopPositionLoop
			} else {
				cNodePosition = types.Position{Column: cNodePosition.Column, Row: cNodePosition.Row - 1}
				cNode = board.Board[cNodePosition.Row][cNodePosition.Column]
			}
		case "bottom":
			if checkRobotOnNode(boardState, types.Position{Column: cNodePosition.Column, Row: cNodePosition.Row + 1}) {
				break stopPositionLoop
			} else {
				cNodePosition = types.Position{Column: cNodePosition.Column, Row: cNodePosition.Row + 1}
				cNode = board.Board[cNodePosition.Row][cNodePosition.Column]
			}
		}
	}
	return cNode, cNodePosition
	//return helper.ConvBytePositionToPosition(cNode)
}

func checkRobotOnNode(boardState types.BoardState, position types.Position) bool {
	robots := helper.SeparateRobots(boardState)

	for _, robot := range robots {
		if position == helper.ConvBytePositionToPosition(robot) {
			return true
		}
	}

	return false
}

func moveRobot(robots [4]byte, robotIndex uint8, endPosition types.Position) (newRobots [4]byte) {
	newRobots = robots
	cRobot := &(newRobots[robotIndex])

	helper.ConvPosToByte(cRobot, endPosition.Column, endPosition.Row)
	return newRobots
}

func isBoardStateInOpenSet(openSet priorityQueue.PriorityQueue, boardState types.BoardState) bool {
	for _, iterateBoardState := range openSet {
		if iterateBoardState.Value == boardState {
			return true
		}
	}
	return false
}

func isBoardStateInClosedSet(closedSet *[]types.BoardState, boardState types.BoardState) bool {
	for _, iterateBoardState := range *closedSet {
		if iterateBoardState == boardState {
			return true
		}
	}
	return false
}

func isRobotOnTarget(boardState *types.BoardState, target uint16) bool {
	targetPosition := helper.ConvBytePositionToPosition(byte(target & 255))
	activeRobotPosition := helper.ConvBytePositionToPosition(byte((*boardState & (255 << 24)) >> 24))
	return activeRobotPosition == targetPosition
}
