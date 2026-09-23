package utils

import (
	"efr_bot/models"
	"sync"
)

var (
	UserStateStorage = make(map[int64]models.UserState)
	mu sync.Mutex
)

func UpdateUserStateStorage(id int64, newState models.UserState) {
	mu.Lock()
	UserStateStorage[id] = newState
	mu.Unlock()
}

func GetUserState(id int64) models.UserState {
	mu.Lock()
	state := UserStateStorage[id]
	mu.Unlock()

	return state
}
