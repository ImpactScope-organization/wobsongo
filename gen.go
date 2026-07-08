//go:generate go tool moq -out internal/mockrepo/document.go -pkg mockrepo internal/data DocumentRepoer
//go:generate go tool sqlc generate

package main
