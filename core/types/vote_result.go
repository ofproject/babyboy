package types

import "github.com/babyboy/common"

type VoteResult struct {
	StartTime       int64          `json:"time_stamp"`
	EndTime         int64          `json:"time_stamp"`
	VoteResult      common.Address `json:"vote_result"`
	ReplacedWitness common.Address `json:"replaced_witness"`
	Round           int64          `json:"round"`
}

func NewVoteResult(startTime int64, endTime int64, voteResult common.Address, replaceWitness common.Address, round int64) VoteResult {
	return VoteResult{startTime, endTime, voteResult, replaceWitness, round}
}
