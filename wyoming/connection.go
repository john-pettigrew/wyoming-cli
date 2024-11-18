package wyoming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

var DescribeMessageType string = "describe"

type WyomingMessage struct {
	Type          string      `json:"type"`
	Version       string      `json:"version"`
	Data          interface{} `json:"data"`
	DataLength    int         `json:"data_length,omitempty"`
	PayloadLength int         `json:"payload_length,omitempty"`
}

type WyomingResponse struct {
	Message WyomingMessage
	Data    []byte
	Payload []byte
}

type WyomingConnection struct {
	Conn          net.Conn
	VoiceServices WyomingVoiceServicesData
}

type WyomingAttribution struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type WyomingVoiceServicesData struct {
	TTS []WyomingVoiceServicesTTSData `json:"tts,omitempty"`
}

// Disconnect disconnects from the Wyoming server.
func (w *WyomingConnection) Disconnect() error {
	return w.Conn.Close()
}

// GetAvailableServices returns the voice services reported to be supported by the
// Wyoming server.
func (w *WyomingConnection) GetAvailableServices() (WyomingVoiceServicesData, error) {
	w.SendMessage(WyomingMessage{Type: DescribeMessageType})

	reader := bufio.NewReaderSize(w.Conn, 1024*1024)
	newMsg, err := w.ReceiveMessageUsingReader(reader)
	if err != nil {
		return WyomingVoiceServicesData{}, err
	}

	var voiceServices WyomingVoiceServicesData
	err = json.Unmarshal(newMsg.Data, &voiceServices)
	if err != nil {
		return WyomingVoiceServicesData{}, err
	}

	return voiceServices, nil
}

// SendMessage sends a message to a Wyoming server followed by a newline character.
func (w *WyomingConnection) SendMessage(msg WyomingMessage) error {
	jsonMessage, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	fmt.Fprintf(w.Conn, "%s\n", string(jsonMessage))
	return nil
}

// ReceiveMessage receives a message from a Wyoming server.
func (w *WyomingConnection) ReceiveMessage() (WyomingResponse, error) {
	reader := bufio.NewReader(w.Conn)
	return w.ReceiveMessageUsingReader(reader)
}

// ReceiveMessageUsingReader receives a message from a Wyoming server using reader. This can
// be helpful if multiple messages are being read or if a larger buffer size is needed.
func (w *WyomingConnection) ReceiveMessageUsingReader(reader *bufio.Reader) (WyomingResponse, error) {
	res := WyomingResponse{Message: WyomingMessage{}}

	msgStr, err := reader.ReadString('\n')
	if err != nil {
		return WyomingResponse{}, err
	}

	err = json.Unmarshal([]byte(msgStr), &res.Message)
	if err != nil {
		return WyomingResponse{}, err
	}

	if res.Message.DataLength > 0 {
		messageData, err := reader.Peek(res.Message.DataLength)
		if err != nil {
			return WyomingResponse{}, err
		}

		res.Data = make([]byte, len(messageData))
		copy(res.Data, messageData)

		_, err = reader.Discard(res.Message.DataLength)
		if err != nil {
			return WyomingResponse{}, err
		}
	}

	if res.Message.PayloadLength > 0 {
		payloadData, err := reader.Peek(res.Message.PayloadLength)
		if err != nil {
			return WyomingResponse{}, err
		}

		res.Payload = make([]byte, len(payloadData))
		copy(res.Payload, payloadData)

		_, err = reader.Discard(res.Message.PayloadLength)
		if err != nil {
			return WyomingResponse{}, err
		}
	}

	return res, nil
}

// Connect connects to a Wyoming server and checks supported features.
func Connect(serverAddr string) (WyomingConnection, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return WyomingConnection{}, err
	}

	w := WyomingConnection{Conn: conn}
	w.VoiceServices, err = w.GetAvailableServices()
	if err != nil {
		return WyomingConnection{}, err
	}

	return w, nil
}
