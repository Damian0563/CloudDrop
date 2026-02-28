package main

type receiveResponse struct {
	Status       string `json:"status"`
	Error        string `json:"error"`
	Msg          string `json:"msg"`
	OriginalName string `json:"original_name"`
	IsDir        bool   `json:"is_dir"`
}

type sendResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}
