package service

import (
	"os/user"
)

type UserInfo struct {
	Username string `json:"username"`
	UID      string `json:"uid"`
	GID      string `json:"gid"`
}


type UserInfoService struct {}

func NewUserInfoService() *UserInfoService {
	return &UserInfoService{}
}

func (s *UserInfoService) GetUserGroupInfo(username string) (*UserInfo, error) {

	usr, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}

	info := &UserInfo{
		Username: username,
		UID:      usr.Uid,
		GID:      usr.Gid,
	}

	return info, nil
}