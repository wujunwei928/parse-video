package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// MCPServer represents the MCP server for video parsing
type MCPServer struct {
	server *server.MCPServer
}

// NewMCPServer creates a new MCP server for video parsing
func NewMCPServer() *MCPServer {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"parse-video",
		"1.0.0",
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Register tools and resources
	RegisterTools(mcpServer)
	RegisterResources(mcpServer)

	return &MCPServer{
		server: mcpServer,
	}
}

// Start starts the MCP server
func (s *MCPServer) Start() error {
	log.Println("Starting MCP Video Parser Server...")
	
	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server
	go func() {
		if err := server.ServeStdio(s.server); err != nil {
			log.Printf("MCP server error: %v", err)
		}
	}()

	log.Println("MCP Video Parser Server is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down MCP server...")
	
	log.Println("MCP server stopped.")
	return nil
}

// Stop stops the MCP server
func (s *MCPServer) Stop() {
	// The server will be stopped when the context is cancelled
	log.Println("MCP server stopping...")
}

// GetServer returns the underlying MCP server
func (s *MCPServer) GetServer() *server.MCPServer {
	return s.server
}

// RunMCPServer creates and starts the MCP server
func RunMCPServer() error {
	mcpServer := NewMCPServer()
	return mcpServer.Start()
}

// RunMCPServerWithStdio creates and starts the MCP server with stdio transport
func RunMCPServerWithStdio() error {
	mcpServer := NewMCPServer()
	
	log.Println("Starting MCP Video Parser Server with stdio transport...")
	
	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server with stdio transport
	go func() {
		if err := server.ServeStdio(mcpServer.server); err != nil {
			log.Printf("MCP stdio server error: %v", err)
		}
	}()

	log.Println("MCP Video Parser Server with stdio transport is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down MCP server...")
	
	log.Println("MCP server stopped.")
	return nil
}

// RunMCPServerWithSSE creates and starts the MCP server with SSE transport
func RunMCPServerWithSSE(port int) error {
	mcpServer := NewMCPServer()
	sseServer := server.NewSSEServer(mcpServer.server)
	
	log.Printf("Starting MCP Video Parser Server with SSE transport on port %d...", port)
	
	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server with SSE transport
	go func() {
		if err := sseServer.Start(fmt.Sprintf(":%d", port)); err != nil {
			log.Printf("MCP SSE server error: %v", err)
		}
	}()

	log.Printf("MCP Video Parser Server with SSE transport is running on port %d. Press Ctrl+C to stop.", port)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down MCP server...")
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sseServer.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	
	log.Println("MCP server stopped.")
	return nil
}