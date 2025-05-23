package main

import (
	"testing"
)

func TestFunctionForPRAnalysis(t *testing.T) {
	testCases := []struct {
		name     string
		a, b, c  int
		inName   string
		expected string
	}{
		{"condition met", 11, 4, 1, "test1", "Condition met for test1"},
		{
			name: "b_is_large",
			a:    1, b: 101, c: 1,
			inName:   "test2",
			expected: "B is large for test2",
		},
		{"default path", 1, 2, 3, "test3", "Default path"},
		{"name empty condition met", 11, 4, 1, "", "Default path"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := FunctionForPRAnalysis(tc.a, tc.b, tc.c, tc.inName)
			if actual != tc.expected {
				t.Errorf("FunctionForPRAnalysis(%d, %d, %d, %s) = %s; want %s",
					tc.a, tc.b, tc.c, tc.inName, actual, tc.expected)
			}
		})
	}
}
