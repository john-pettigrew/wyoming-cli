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
	Conn net.Conn
}

type WyomingAttribution struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type WyomingVoiceServicesData struct {
	TTS []WyomingVoiceServicesTTSData `json:"tts,omitempty"`
}

func (w *WyomingConnection) Disconnect() error {
	return w.Conn.Close()
}

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

func (w *WyomingConnection) SendMessage(msg WyomingMessage) error {
	jsonMessage, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	fmt.Fprintf(w.Conn, "%s\n", string(jsonMessage))
	return nil
}

func (w *WyomingConnection) ReceiveMessage() (WyomingResponse, error) {
	reader := bufio.NewReader(w.Conn)
	return w.ReceiveMessageUsingReader(reader)
}

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

func Connect(serverAddr string) (WyomingConnection, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return WyomingConnection{}, err
	}

	return WyomingConnection{Conn: conn}, nil
}
