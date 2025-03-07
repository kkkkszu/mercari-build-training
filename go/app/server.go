package app

import (
    "crypto/sha256"
    "encoding/json"
    "log/slog"
    "encoding/hex"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)

type Server struct {
	// Port is the port number to listen on.
	Port string
	// ImageDirPath is the path to the directory storing images.
	ImageDirPath string
}

// Run is a method to start the server.
// This method returns 0 if the server started successfully, and 1 otherwise.
func (s Server) Run() int {
	// set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	// STEP 4-6: set the log level to DEBUG
	slog.SetLogLoggerLevel(slog.LevelInfo)

	// set up CORS settings
	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// STEP 5-1: set up the database connection

	// set up handlers
	itemRepo := NewItemRepository()
	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// set up routes
	 mux := http.NewServeMux()
         // POST /items - 商品追加
         // /items エンドポイント
mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodPost:
        h.AddItem(w, r)  // POST /items - 商品追加
    case http.MethodGet:
        h.GetItems(w, r)  // GET /items - 商品一覧取得
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
})
    //mux.HandleFunc("/items", h.AddItem)  // GET /items
    //mux.HandleFunc("/items", h.GetItems)   // POST /items
    mux.HandleFunc("/images/{filename}", h.GetImage) // GET /images/{filename}

	// start the server
	slog.Info("http server started on", "port", s.Port)
	err := http.ListenAndServe(":"+s.Port, simpleCORSMiddleware(simpleLoggerMiddleware(mux), frontURL, []string{"GET", "HEAD", "POST", "OPTIONS"}))
	if err != nil {
		slog.Error("failed to start server: ", "error", err)
		return 1
	}

	return 0
}

type Handlers struct {
	// imgDirPath is the path to the directory storing images.
	imgDirPath string
	itemRepo   ItemRepository
}

type HelloResponse struct {
	Message string `json:"message"`
}

// Hello is a handler to return a Hello, world! message for GET / .
func (s *Handlers) Hello(w http.ResponseWriter, r *http.Request) {
	resp := HelloResponse{Message: "Hello, world!"}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type AddItemRequest struct {
	Name string `form:"name"`
	Category string `form:"category"` // STEP 4-2: add a category field
	Image []byte `form:"image"` // STEP 4-4: add an image field
        
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the request to add an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, error) {
	req := &AddItemRequest{
		Name: r.FormValue("name"),
		Category: r.FormValue("category"),//step4-2:Category取得
	}

	// STEP 4-4: add an image field

	// validate the request
	if req.Name == "" || req.Category == ""{
		return nil, errors.New("name is required")
	}
	// STEP 4-2: validate the category field (|| req.Category == "")

	// STEP 4-4: validate the image fiel
// AddItem is a handler to add a new item for POST /items .
   return req, nil
}

// AddItem is a handler to add a new item for POST /items .
func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Parse form data
    err := r.ParseMultipartForm(10 << 20) // 10 MB limit for image upload
    if err != nil {
        http.Error(w, "Unable to parse form", http.StatusBadRequest)
        return
    }

    req, err := parseAddItemRequest(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }


    // Handle image if provided
    var imageFileName string
    file, _, err := r.FormFile("image")
    if err == nil {
        // Read the image file into []byte
        fileData, err := ioutil.ReadAll(file)
        if err != nil {
            http.Error(w, "Unable to read image", http.StatusInternalServerError)
            return
        }

        // Store the image and get the file name
        imageFileName, err = s.storeImage(fileData) // Ensure fileData is passed as []byte
        if err != nil {
            slog.Error("failed to store image: ", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }

    // Create the item object with image file name
    item := &Item{
        Name:      req.Name,
        Category:  req.Category,
        ImageName: imageFileName, // Store image file name
    }

    // Save the item to the repository
    err = s.itemRepo.Insert(ctx, item)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Respond with success message
    message := fmt.Sprintf("item received: %s", item.Name)
    resp := AddItemResponse{Message: message}
    err = json.NewEncoder(w).Encode(resp)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}
      // 商品リストを取得
func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {

	items, err := s.itemRepo.GetItems(r.Context())
	if err != nil {
		http.Error(w, "failed to fetch items", http.StatusInternalServerError)
		return
	}

	// レスポンスとして商品リストをJSONで返す
	resp := struct {
		Items []Item `json:"items"`
	}{
		Items: items,
	}

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// storeImage stores an image and returns the file path and an error if any.
// this method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
// storeImage stores an image and returns the file path and an error if any.
// This method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
func (s *Handlers) storeImage(image []byte) (filePath string, err error) {
    // Create an image directory if it does not exist
    imageDir := "images" // Image directory path
    err = os.MkdirAll(imageDir, os.ModePerm) // Create the directory if it doesn't exist
    if err != nil {
        return "", fmt.Errorf("failed to create image directory: %v", err)
    }

    // Calculate SHA-256 hash of the image
    hash := sha256.New()
    _, err = hash.Write(image)
    if err != nil {
        return "", fmt.Errorf("failed to hash image: %v", err)
    }

    // Convert the hash to a hexadecimal string
    hashBytes := hash.Sum(nil)
    fileName := hex.EncodeToString(hashBytes) + ".jpg" // Generate the file name with .jpg extension

    // Define the full path where the image will be stored
    filePath = filepath.Join(imageDir, fileName)

    // Check if the file already exists
    if _, err := os.Stat(filePath); err == nil {
        return filePath, nil // Return the existing file path if file already exists
    }

    // Write the image to the file
    err = ioutil.WriteFile(filePath, image, 0644)
    if err != nil {
        return "", fmt.Errorf("failed to store image: %v", err)
    }

    return filePath, nil
}


type GetImageRequest struct {
	FileName string // path value
}

// parseGetImageRequest parses and validates the request to get an image.
func parseGetImageRequest(r *http.Request) (*GetImageRequest, error) {
	req := &GetImageRequest{
		FileName: r.PathValue("filename"), // from path parameter
	}

	// validate the request
	if req.FileName == "" {
		return nil, errors.New("filename is required")
	}

	return req, nil
}

// GetImage is a handler to return an image for GET /images/{filename} .
// If the specified image is not found, it returns the default image.
func (s *Handlers) GetImage(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetImageRequest(r)
	if err != nil {
		slog.Warn("failed to parse get image request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imgPath, err := s.buildImagePath(req.FileName)
	if err != nil {
		if !errors.Is(err, errImageNotFound) {
			slog.Warn("failed to build image path: ", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// when the image is not found, it returns the default image without an error.
		slog.Debug("image not found", "filename", imgPath)
		imgPath = filepath.Join(s.imgDirPath, "default.jpg")
	}

	slog.Info("returned image", "path", imgPath)
	http.ServeFile(w, r, imgPath)
}

// buildImagePath builds the image path and validates it.
func (s *Handlers) buildImagePath(imageFileName string) (string, error) {
	imgPath := filepath.Join(s.imgDirPath, filepath.Clean(imageFileName))

	// to prevent directory traversal attacks
	rel, err := filepath.Rel(s.imgDirPath, imgPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid image path: %s", imgPath)
	}

	// validate the image suffix
	if !strings.HasSuffix(imgPath, ".jpg") && !strings.HasSuffix(imgPath, ".jpeg") {
		return "", fmt.Errorf("image path does not end with .jpg or .jpeg: %s", imgPath)
	}

	// check if the image exists
	_, err = os.Stat(imgPath)
	if err != nil {
		return imgPath, errImageNotFound
	}

	return imgPath, nil
}
