package callbacks

// SNAPResponse adalah bentuk umum balasan ke provider (format final ditentukan adapter provider).
type SNAPResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}
