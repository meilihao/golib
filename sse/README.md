# README
```go
// main.go: use gin
xxx.POST("/sse", svc.HandleSSE(svc.SSEManager))

// svc/sse.go
var (
	SSEManager = sse.NewManager[string, int]()
)

// sse推送
func HandleSSE(manager *sse.Manager[string, int]) gin.HandlerFunc {
	return func(c *gin.Context) {
		bearer := strings.TrimSpace(strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Bearer "))
		if bearer == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client"})
			return
		}

		// Set SSE headers
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no") // Disable buffering for Nginx

		// 避免chrome f12 sse请求不显示"EventStream"
		fmt.Fprintf(c.Writer, "data: %s\n\n", fmt.Sprintf(`{"event": "time", "data":"%s"}`, time.Now().Format(time.DateTime)))
		c.Writer.Flush()

		accountStr := c.Query("operator_id")
		account, _ := strconv.Atoi(accountStr)

		client := &sse.Client[string, int]{
			Id:      bearer,
			Account: account,
			Channel: make(chan *sse.Data[int], 8),
		}

		// 避免重复连接
		if manager.HasClient(client) {
			c.JSON(http.StatusConflict, gin.H{"error": "client id already exists"})
			return
		}

		manager.Register <- client

		defer func() {
			manager.Unregister <- client
		}()

		closeNotify := c.Writer.CloseNotify()

		for {
			select {
			case data, ok := <-client.Channel:
				log.Glog.Debug("received message", zap.Any("data", data))

				if !ok {
					return
				}

				// Write the SSE formatted message
				// Format: "event: message\ndata: {\"message\":\"...\"}\n\n"
				if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(data.Message)); err != nil {
					log.Glog.Error("Error send SSE data", zap.Error(err))
					return
				}

				// Flush the data immediately to send it to the client
				c.Writer.Flush()
			case <-closeNotify:
				return
			}
		}
	}
}
```