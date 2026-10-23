// Package httpx provides a small, uniform JSON response envelope for handlers.
package httpx

import "github.com/gofiber/fiber/v2"

// Response is the standard API envelope.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// OK writes a 200 success response.
func OK(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusOK).JSON(Response{Success: true, Data: data})
}

// Created writes a 201 success response.
func Created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(Response{Success: true, Data: data})
}

// Err writes a failure response with the given status code and message.
func Err(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(Response{Success: false, Error: msg})
}
