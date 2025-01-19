package contracts

type QueueMessage struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	To      string `json:"to"`
	Retry   int    `json:"retry"`
}