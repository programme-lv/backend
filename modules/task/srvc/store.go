package srvc

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime"
	"path"
	"path/filepath"
	"strings"

	"github.com/programme-lv/backend/common/filestore"
	"github.com/programme-lv/backend/common/srvcerror"
)

const taskIllustrationDir = "illustrations"

const taskStatementImageDir = "md-images"

func taskIllustrationObjectKey(storedKey string) string {
	if storedKey == "" {
		return ""
	}
	if strings.Contains(storedKey, "/") {
		return storedKey
	}
	return path.Join(taskIllustrationDir, storedKey)
}

func taskIllustrationStoredKey(objectKey string) string {
	return strings.TrimPrefix(objectKey, taskIllustrationDir+"/")
}

func taskStatementImageObjectKey(storedKey string) string {
	if storedKey == "" {
		return ""
	}
	if strings.HasPrefix(storedKey, taskStatementImageDir+"/") ||
		strings.HasPrefix(storedKey, "task/") ||
		strings.HasPrefix(storedKey, "task-md-images/") {
		return storedKey
	}
	return path.Join(taskStatementImageDir, storedKey)
}

func taskStatementImageStoredKey(objectKey string) string {
	return strings.TrimPrefix(objectKey, taskStatementImageDir+"/")
}

// UploadIllustrationImg stores an illustration and returns the stored key without the illustrations/ prefix.
// The object key is illustrations/<sha256>.<ext>.
func (ts *taskSrvc) UploadIllustrationImg(ctx context.Context, mimeType string, body []byte) (string, srvcerror.E) {
	l := ts.logger(ctx)
	sha2 := sha2Hex(body)
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", errUnknownImageExt(mimeType)
	}
	if len(exts) == 0 {
		return "", errUnknownImageExt(mimeType)
	}
	ext := exts[0]
	storedKey := fmt.Sprintf("%s%s", sha2, ext)
	_, err = ts.publicStore.Upload(body, taskIllustrationObjectKey(storedKey), mimeType)
	if err != nil {
		l.Error("upload illustration", "error", err)
		return "", srvcerror.InternalServerError()
	}
	return storedKey, nil
}

// UploadStatementImage stores a statement image and records it on the task.
// The object key is md-images/<taskId>/<sha256-prefix>.<ext>.
func (ts *taskSrvc) UploadStatementImage(ctx context.Context, taskId string, imgFilename string, imageMimeType string, body []byte) (string, srvcerror.E) {
	l := ts.logger(ctx)

	ext, err := mimeToFileExt(imageMimeType)
	if err != nil {
		return "", errUnknownImageExt(imageMimeType)
	}

	width, height, err := getImgWidthHeighPx(body, imageMimeType)
	if err != nil {
		return "", ErrImageDimensions
	}

	if width > 2000 || height > 2000 || width == 0 || height == 0 {
		return "", ErrImageSize
	}

	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("get corresponding task", "error", err)
		return "", srvcerror.InternalServerError()
	}
	for _, img := range t.MdImages {
		if img.Filename == imgFilename {
			return "", errImageAlreadyExists(imgFilename)
		}
	}

	storedKey := fmt.Sprintf("%s/%s%s", taskId, sha2Hex(body)[:12], ext)
	objectURI, err := ts.publicStore.Upload(body, taskStatementImageObjectKey(storedKey), imageMimeType)
	if err != nil {
		l.Error("upload statement image", "error", err)
		return "", srvcerror.InternalServerError()
	}

	err = ts.repo.AddStatementImg(ctx, taskId, StatementImage{
		ObjectKey: storedKey,
		Filename:  imgFilename,
		WidthPx:   width,
		HeightPx:  height,
		SzInBytes: len(body),
	})
	if err != nil {
		l.Error("add statement image to db", "error", err)
		return "", srvcerror.InternalServerError()
	}
	return objectURI, nil
}

func (ts *taskSrvc) DeleteStatementImage(ctx context.Context, taskId string, filename string) srvcerror.E {
	l := ts.logger(ctx)

	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("get task", "error", err)
		return srvcerror.InternalServerError()
	}

	var targetImage *StatementImage
	for _, img := range t.MdImages {
		if img.Filename == filename {
			targetImage = &img
			break
		}
	}

	if targetImage == nil {
		return errImageNotFound(filename)
	}

	objectKey := taskStatementImageObjectKey(targetImage.ObjectKey)

	exists, err := ts.publicStore.Exists(objectKey)
	if err != nil {
		l.Error("check if statement image exists", "error", err)
		return srvcerror.InternalServerError()
	}
	if !exists {
		l.Error("statement image missing from store", "object_key", objectKey)
		return srvcerror.InternalServerError()
	}

	err = ts.repo.DeleteStatementImg(ctx, taskId, filename)
	if err != nil {
		l.Error("delete image from database", "error", err)
		return srvcerror.InternalServerError()
	}

	err = ts.publicStore.Delete(objectKey)
	if err != nil {
		l.Error("delete statement image from store", "error", err)
		return srvcerror.InternalServerError()
	}

	return nil
}

func (ts *taskSrvc) DeleteIllustrationImg(ctx context.Context, taskId string) srvcerror.E {
	l := ts.logger(ctx)

	t, err := ts.repo.GetTask(ctx, taskId)
	if err != nil {
		l.Error("get task", "error", err)
		return srvcerror.InternalServerError()
	}

	if t.IllustrImg.ObjectKey == "" {
		return ErrIllustrationNotFound
	}

	objectKey := taskIllustrationObjectKey(t.IllustrImg.ObjectKey)

	exists, err := ts.publicStore.Exists(objectKey)
	if err != nil {
		l.Error("check if illustration exists", "error", err)
		return srvcerror.InternalServerError()
	}
	if !exists {
		l.Error("illustration missing from store", "object_key", objectKey)
		return srvcerror.InternalServerError()
	}

	emptyImg := IllustrationImage{
		ObjectKey: "",
		WidthPx:   0,
		HeightPx:  0,
		SzInBytes: 0,
	}
	err = ts.repo.UpdateIllustrationImg(ctx, taskId, emptyImg)
	if err != nil {
		l.Error("update illustration image in database", "error", err)
		return srvcerror.InternalServerError()
	}

	err = ts.publicStore.Delete(objectKey)
	if err != nil {
		l.Error("delete illustration from store", "error", err)
		return srvcerror.InternalServerError()
	}

	return nil
}

// UpdateIllustrationImg writes illustration metadata to the database.
// The stored object key is saved without the illustrations/ prefix.
func (ts *taskSrvc) UpdateIllustrationImg(ctx context.Context, taskId string, img IllustrationImage) srvcerror.E {
	l := ts.logger(ctx)
	img.ObjectKey = taskIllustrationStoredKey(img.ObjectKey)

	err := ts.repo.UpdateIllustrationImg(ctx, taskId, img)
	if err != nil {
		l.Error("update illustration image in database", "error", err)
		return srvcerror.InternalServerError()
	}

	return nil
}

func (ts *taskSrvc) GetHttpUrlForIllustrImg(ctx context.Context, objectKey string) (string, srvcerror.E) {
	url, err := filestore.AssetURL(ts.apiPublicBaseURL, taskIllustrationObjectKey(objectKey))
	if err != nil {
		ts.logger(ctx).Error("build illustration image URL", "error", err)
		return "", srvcerror.InternalServerError()
	}
	return url, nil
}

func (ts *taskSrvc) GetHttpUrlForStatementImage(ctx context.Context, objectKey string) (string, srvcerror.E) {
	url, err := filestore.AssetURL(ts.apiPublicBaseURL, taskStatementImageObjectKey(objectKey))
	if err != nil {
		ts.logger(ctx).Error("build statement image URL", "error", err)
		return "", srvcerror.InternalServerError()
	}
	return url, nil
}

func mimeToFileExt(mimeType string) (string, error) {
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", fmt.Errorf("get file extension: %w", err)
	}
	if len(exts) == 0 {
		return "", fmt.Errorf("file extension not found")
	}
	return exts[0], nil
}

func mimeFromFname(fname string) (string, error) {
	ext := filepath.Ext(fname)
	if ext == "" {
		return "", fmt.Errorf("file extension not found")
	}
	return mime.TypeByExtension(ext), nil
}

func getImgWidthHeighPx(body []byte, mimeType string) (int, int, error) {
	reader := bytes.NewReader(body)

	switch mimeType {
	case "image/png":
		img, err := png.Decode(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("decode PNG image: %w", err)
		}
		return img.Bounds().Dx(), img.Bounds().Dy(), nil
	case "image/jpeg", "image/jpg":
		config, err := jpeg.DecodeConfig(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("decode JPEG image: %w", err)
		}
		return config.Width, config.Height, nil
	default:
		config, _, err := image.DecodeConfig(reader)
		if err != nil {
			return 0, 0, fmt.Errorf("decode image: %w", err)
		}
		return config.Width, config.Height, nil
	}
}
