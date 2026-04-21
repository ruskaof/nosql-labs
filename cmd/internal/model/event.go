package model

type CreateEventRequest struct {
	Title       *string `json:"title"`
	Address     *string `json:"address"`
	StartedAt   *string `json:"started_at"`
	FinishedAt  *string `json:"finished_at"`
	Description *string `json:"description"`
}

type PatchEventRequest struct {
	Category *string `json:"category"`
	Price    *uint64 `json:"price"`
	City     *string `json:"city"`
}
