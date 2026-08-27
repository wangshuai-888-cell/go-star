package main

import (
	"encoding/json"
	"fmt"
)

type RemoveRequest struct {
	IDList []uint `json:"idList"`
}

func main() {
	cases := []string{
		`{"idList":[59,60]}`,
		`{"idList":[59,"60"]}`,
		`{"idList":[59,"abc"]}`,
		`{"idList":["59","60"]}`,
		`{"idList":[59,null]}`,
	}
	for _, body := range cases {
		var cr RemoveRequest
		err := json.Unmarshal([]byte(body), &cr)
		fmt.Printf("body=%s\n  err=%v\n  IDList=%v\n\n", body, err, cr.IDList)
	}
}
