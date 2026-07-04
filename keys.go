package main

import "strings"

// parseKeys 는 "diw", "cw bye<esc>" 같은 입력 문자열을 Key 시퀀스로 변환한다.
// 특수키 토큰: <esc> <cr> <bs> <c-r>. par 산출(레벨의 Solution 길이 계산)과
// 테스트의 feedKeys() 양쪽에서 공유하는 프로덕션 파서.
func parseKeys(s string) []Key {
	var out []Key
	for i := 0; i < len(s); {
		if s[i] == '<' {
			if j := strings.IndexByte(s[i:], '>'); j >= 0 {
				tok := s[i+1 : i+j]
				switch tok {
				case "esc", "cr", "bs", "c-r":
					out = append(out, SpecialKey(tok))
					i += j + 1
					continue
				}
			}
		}
		out = append(out, RuneKey(rune(s[i])))
		i++
	}
	return out
}
