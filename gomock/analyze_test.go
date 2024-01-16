package gomock_test

import (
	"fmt"
	"testing"
)

func TestNewGoMock(t *testing.T) {
	testcases := []struct {
		name string
	}{{name: ""}}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			fmt.Println("write your unit test!")
		})
	}
}
