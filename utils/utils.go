package utils

import (
	"fmt"
	"regexp"
)

var escapedSchemeURLRegexp = regexp.MustCompile(`(?i)(https?)\\://`)

func RegexpMatchUrlFromString(str string) (string, error) {
	str = escapedSchemeURLRegexp.ReplaceAllString(str, `${1}://`)
	urlReg, err := regexp.Compile(`(?i)https?://[\w.%:@\[\]-]+[\w/-]*[\w.-:]*\??[\w=&:\-+%.]*/*`)
	if err != nil {
		return "", fmt.Errorf("match url regexp compile error: %s", err.Error())
	}

	findStr := urlReg.FindString(str)
	if len(findStr) <= 0 {
		return "", fmt.Errorf("str not have url")
	}

	return findStr, nil
}
