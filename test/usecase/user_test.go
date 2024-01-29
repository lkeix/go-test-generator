package usecase_test

import (
	"fmt"
	"testing"
)

func TestNewUser(t *testing.T) {
	testcases := []struct {
		name string
	}{{name: ""}}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			fmt.Println("write your unit test!")
		})
	}
}
func Testuser_Create(t *testing.T) {
	testcases := []struct {
		name string
	}{{name: ""}, {name: ""}}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			fmt.Println("write your unit test!")
		})
	}
}
