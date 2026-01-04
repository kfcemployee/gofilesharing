package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	pb "github.com/kfcempoyee/gofilesharing/gen/registry/proto/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FileHandler struct {
	TmpDir     string
	GRpcClient pb.RegServiceClient
	Logger     *slog.Logger
}

type ErrorResponse struct {
	Error string
}

func handleError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(
		ErrorResponse{Error: msg},
	)
}

const idRegexp = `^[a-zA-Z0-9]+$`

func (h *FileHandler) fetchFile(w http.ResponseWriter, r *http.Request) *pb.GetFileDataResp {
	path := strings.TrimSpace(r.PathValue("id"))
	if ok, _ := regexp.MatchString(idRegexp, path); !ok {
		h.Logger.Error("request not handled: invalid link.")
		handleError(w, "File link should contain only letters and digits.", http.StatusBadRequest)
		return nil
	}

	resp, err := h.GRpcClient.GetFile(r.Context(), &pb.GetFileDataReq{ShortName: path})
	if err != nil {

		st, ok := status.FromError(err)
		if !ok {
			h.Logger.Error("failed to call rpc.", "details", err)
			handleError(w, "Server Error.", http.StatusInternalServerError)
			return nil
		}

		h.Logger.Error("rpc error",
			"code", st.Code(),
			"msg", st.Message(),
			"details", st.Details(),
		)

		switch st.Code() {
		case codes.DeadlineExceeded:
			handleError(w, "Link is not valid or expired.", http.StatusNotFound)
		case codes.NotFound:
			handleError(w, "File not found.", http.StatusNotFound)
		default:
			handleError(w, "Service Internal error.", http.StatusInternalServerError)
		}

		return nil
	}

	return resp
}

func (h *FileHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Not Found", http.StatusMethodNotAllowed)
		return
	}

	resp := h.fetchFile(w, r)
	if resp == nil {
		return
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+resp.Filename+"\"")
	http.ServeFile(w, r, resp.StPath)
}

func (h *FileHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	resp := h.fetchFile(w, r)
	if resp == nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Name        string
		Size        int
		ContentType string
	}{
		Name:        resp.Filename,
		Size:        int(resp.SizeBytes),
		ContentType: resp.ContentType,
	})
}

func (h *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	const maxUploadSize = 32 << 20 // 32 кБ
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	reader, err := r.MultipartReader()
	if err != nil {
		handleError(w, "Invalid Multipart Form.", http.StatusInternalServerError)
		h.Logger.Error("invalid multipart form")
		return
	}

	for {
		part, err := reader.NextPart()

		// когда закончился поток, выходим из цикла.
		if err == io.EOF {
			break
		}

		if err != nil {
			h.Logger.Error("failed to read data", "details", err)
			handleError(w, "Failed to upload a file.", http.StatusInternalServerError)
			return
		}

		// файл должен лежать в поле File формы
		if part.FormName() == "File" {

			// для определения типа файла читаем первые 512 байт файла (сигнатуру)
			snBuff := make([]byte, 512)

			n, _ := part.Read(snBuff)
			// тут определяем тип контента и кладём в переменную
			cType := http.DetectContentType(snBuff[:n])

			if cType == "application/ms-executable" {
				handleError(w, "This type of files is not available.", http.StatusInternalServerError)
				return
			}

			tmpName := uuid.New().String()
			tmpDir, _ := os.Create(filepath.Join(h.TmpDir, tmpName))

			// склеиваем буфер с первыми байтами и следующую часть
			fullReader := io.MultiReader(bytes.NewReader(snBuff[:n]), part)

			size, _ := io.Copy(tmpDir, fullReader)
			tmpDir.Close()

			resp, err := h.GRpcClient.RegisterFile(r.Context(), &pb.RegisterFileRequest{
				TmpName:     tmpName,
				Filename:    part.FileName(),
				SizeBytes:   size,
				ContentType: cType,
			})

			if err != nil {
				st, ok := status.FromError(err)

				if !ok {
					h.Logger.Error("failed to call rpc", "detail", err)
					handleError(w, "Server Error.", http.StatusInternalServerError)
					return
				}

				h.Logger.Error(
					"rpc error",
					"code", st.Code(),
					"details", st.Details(),
				)

				switch st.Code() {
				case codes.Internal:
					handleError(w, "Uploading failed due to server error.", http.StatusInternalServerError)
				}

				return
			}

			json.NewEncoder(w).Encode(resp)
		}
	}
}
