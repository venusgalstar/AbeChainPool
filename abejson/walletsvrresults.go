package abejson

type GetBalancesabeResult struct {
	CurrentTime        string  `json:"current_time,omitempty"`
	CurrentHeight      int32   `json:"current_height,omitempty"`
	CurrentBlockHash   string  `json:"current_block_hash,omitempty"`
	TotalBalance       float64 `json:"total_balance"`
	SpendableBalance   float64 `json:"spendable_balance"`
	ImmatureCBBalance  float64 `json:"immature_cb_balance"`
	ImmatureTRBalance  float64 `json:"immature_tr_balance"`
	UnconfirmedBalance float64 `json:"unconfirmed_balance"`
}
