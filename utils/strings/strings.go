package strings

import "regexp"

// NamedString 命名字符串有效性：字母数字下划线组成的，且首字母只能是字母，长度不超过25个字符
func NamedString(src string) bool {
	if len(src) > 25 {
		return false
	}
	reg := regexp.MustCompile("^[a-zA-Z][a-zA-Z_0-9]*$")
	return reg.MatchString(src)
}
