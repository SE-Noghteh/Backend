package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type employerUpload struct {
	kind  string
	file  *ports.UploadFile
	video bool
}

func uploadEmployerFiles(ctx context.Context, storage ports.Storage, media ports.MediaValidator, namespace string, files []employerUpload, uploaded *[]string) (map[string]*valueobjects.StoredObject, error) {
	validated := make(map[string]*ports.ValidatedFile)
	// Validate all files before putting any object in storage.
	for _, spec := range files {
		if spec.file == nil {
			continue
		}
		var file *ports.ValidatedFile
		var err error
		if spec.video {
			file, err = media.Video(ctx, *spec.file)
		} else {
			file, err = media.Document(ctx, *spec.file)
		}
		if err != nil {
			return nil, err
		}
		validated[spec.kind] = file
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	prefix := namespace + "/" + hex.EncodeToString(nonce) + "/"
	objects := make(map[string]*valueobjects.StoredObject)
	for _, spec := range files {
		file := validated[spec.kind]
		if file == nil {
			continue
		}
		key := prefix + spec.kind + file.Extension
		// Include ambiguous PUT failures in compensating cleanup.
		*uploaded = append(*uploaded, key)
		object, err := storage.Put(ctx, key, file.Body, file.ContentType, file.Size)
		if err != nil {
			return nil, domainerrors.ErrStorageUnavailable
		}
		objects[spec.kind] = object
	}
	return objects, nil
}

func cleanupEmployerUploads(ctx context.Context, storage ports.Storage, keys []string) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	for _, key := range keys {
		if err := storage.Delete(cleanupCtx, key); err != nil {
			slog.Error("cleanup employer upload failed", "error", err)
		}
	}
}

func employerDocumentURL(ctx context.Context, storage ports.Storage, object *valueobjects.StoredObject) (string, error) {
	if object == nil {
		return "", domainerrors.ErrInvalidUpload
	}
	url, err := storage.PresignGet(ctx, object.Key, 5*time.Minute)
	if err != nil {
		return "", domainerrors.ErrStorageUnavailable
	}
	return url, nil
}
