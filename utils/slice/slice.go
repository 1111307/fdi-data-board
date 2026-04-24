package slice

import (
	"sort"
	"strings"
)

// ContainsString args: []string, string
func ContainsString(slice []string, item string) bool {
	for _, j := range slice {
		if j == item {
			return true
		}
	}
	return false
}

// FuzzyContainsString args: []string, string
func FuzzyContainsString(slice []string, substr string) []string {
	if len(substr) == 0 {
		return slice
	}

	newSlice := make([]string, 0)
	for _, s := range slice {
		if strings.Contains(s, substr) {
			newSlice = append(newSlice, s)
		}
	}
	return newSlice
}

// StringsEqual args: []string, []string
func StringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// StringEqualSort args: []string, []string 用于比较切片中元素是否都一样
func StringEqualSort(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	sortLeft, sortRight := left, right
	sort.Strings(sortLeft)
	sort.Strings(sortRight)

	return StringsEqual(sortLeft, sortRight)
}

// CompareSlice 比较两个slice，第一个参数返回少了的item，第二个参数返回新的item
func CompareSlice(s1, s2 []string) ([]string, []string) {
	sort.Strings(s1)
	sort.Strings(s2)

	deleteSlice, addSlice := make([]string, 0), make([]string, 0)

	i, j := 0, 0
	for i < len(s1) && j < len(s2) {
		if s1[i] == s2[j] {
			i++
			j++
			continue
		}
		if s1[i] < s2[j] {
			deleteSlice = append(deleteSlice, s1[i])
			i++
			continue
		}
		if s1[i] > s2[j] {
			addSlice = append(addSlice, s2[j])
			j++
			continue
		}
	}
	if len(s1) == i {
		addSlice = append(addSlice, s2[j:]...)
	}
	if len(s2) == j {
		deleteSlice = append(deleteSlice, s1[i:]...)
	}

	return deleteSlice, addSlice
}

// UniqueStringSlice 去重
func UniqueStringSlice(src []string) []string {
	uFilter := make(map[string]struct{})
	for _, s := range src {
		uFilter[s] = struct{}{}
	}

	dst := make([]string, 0, len(uFilter))
	for k, _ := range uFilter {
		dst = append(dst, k)
	}
	return dst
}

// IntersectSliceInt 任意数量slice的交集
func IntersectSliceInt(slicelist ...[]int) []int {
	m := make(map[int]int)
	iss := make([]int, 0)

	for _, s := range slicelist {
		for _, v := range s {
			m[v]++
		}
	}

	for k, v := range m {
		if v == len(slicelist) {
			iss = append(iss, k)
		}
	}
	return iss
}

// IntersectSliceString 任意数量slice的交集
func IntersectSliceString(slicelist ...[]string) []string {
	m := make(map[string]int)
	iss := make([]string, 0)

	for _, s := range slicelist {
		for _, v := range s {
			m[v]++
		}
	}

	for k, v := range m {
		if v == len(slicelist) {
			iss = append(iss, k)
		}
	}
	return iss
}
