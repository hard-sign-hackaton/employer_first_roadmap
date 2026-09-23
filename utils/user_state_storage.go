package utils

import (
	"efr_bot/models"
	"sync"
)

var (
	UserStateStorage = make(map[int64]models.UserState)
	UserStateStorageMutex sync.Mutex
)

func UpdateUserStateStorage(id int64, newState models.UserState) {
	UserStateStorageMutex.Lock()
	UserStateStorage[id] = newState
	UserStateStorageMutex.Unlock()
}

func GetUserState(id int64) models.UserState {
	UserStateStorageMutex.Lock()
	state := UserStateStorage[id]
	UserStateStorageMutex.Unlock()

	return state
}
