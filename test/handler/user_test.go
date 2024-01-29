package handler_test

import (
	"fmt"
	"testing"
)

func TestUser_Create(t *testing.T) {
	testcases := []struct {
		name string
	}{{name: ""}, {name: ""}, {name: ""}}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			fmt.Println("write your unit test!")
		})
	}
}
