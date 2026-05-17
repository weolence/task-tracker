package http

import (
	"errors"
	"io"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

func readProtoJSON(r *http.Request, message proto.Message) error {
	lr := io.LimitReader(r.Body, maxRequestBodyBytes+1)
	body, err := io.ReadAll(lr)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if int64(len(body)) > maxRequestBodyBytes {
		return errors.New("request body too large")
	}

	return protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, message)
}

func writeProtoJSON(w http.ResponseWriter, status int, message proto.Message) {
	b, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(message)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b)
}
