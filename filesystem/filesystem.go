package filesystem

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mholt/archiver"
)

// FSManager represents information for controlling the student's filesystem
type FSManager struct {
	StudentID           string
	Course              string
	StudentFSIdentifier string
}

// Init initializes the AWS session and returns the session
func Init() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewSharedCredentials("", os.Getenv("RHETOR_AWS_PROFILE")),
		Region:      aws.String("us-east-1"),
	})
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// New creates a new FSManager
func New(studentID string, course string) (*FSManager, error) {
	return &FSManager{
		StudentID:           studentID,
		Course:              course,
		StudentFSIdentifier: course + "-" + studentID,
	}, nil
}

// LoadStudentFilesDisk creates the student's filesystem on disk from their
// saved .tgz file in S3. If no .tgz file exists, it downloads the default starter
func (fs *FSManager) LoadStudentFilesDisk(sess *session.Session) error {
	// check to see if the directory exists. If not, make it
	path := "/usr/local/share/rhetor"
	if _, err := os.Stat(path + "/" + fs.StudentFSIdentifier); os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	} else {
		return nil
	}
	// init the downloader
	downloader := s3manager.NewDownloader(sess)
	// create the placeholder file
	f, err := os.Create(path + "/" + fs.StudentFSIdentifier + ".tgz")
	if err != nil {
		return fmt.Errorf("failed to create file %q, %v", fs.StudentFSIdentifier+".tgz", err)
	}
	// Write the contents of S3 Object to the file
	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String("rhetor"),
		Key:    aws.String(fs.StudentFSIdentifier + ".tgz"),
	})
	if err != nil {
		if err.Error()[:9] == "NoSuchKey" {
			n, err = downloader.Download(f, &s3.GetObjectInput{
				Bucket: aws.String("rhetor"),
				Key:    aws.String(fs.Course + "-starter.tgz"),
			})
			if err != nil {
				return fmt.Errorf("failed to download starter file")
			}
		} else {
			return fmt.Errorf("failed to download file, %v", err)
		}
	}
	fmt.Printf("file downloaded, %d bytes\n", n)
	if err := archiver.TarGz.Open(path+"/"+fs.StudentFSIdentifier+".tgz", path); err != nil {
		return fmt.Errorf("failed to extract archive, %v", err)
	}
	os.Remove(path + "/" + fs.StudentFSIdentifier + ".tgz")
	return nil
}

// SaveStudentFilesAWS creates a .tgz file of the student's directory and uploads
// it to S3
func (fs *FSManager) SaveStudentFilesAWS(sess *session.Session) error {
	// init the uploader
	uploader := s3manager.NewUploader(sess)
	filePath := "/usr/local/share/rhetor/" + fs.StudentFSIdentifier
	fileName := filePath + ".tgz"
	// make a .tgz of the student's directory
	if err := archiver.TarGz.Make(fileName, []string{filePath}); err != nil {
		fmt.Println(err.Error())
		return err
	}
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filePath, err)
	}
	// upload to s3
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("rhetor"),
		Key:    aws.String(fs.StudentFSIdentifier + ".tgz"),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v error", err)
	}
	// remove student's folder from disk
	if err := os.RemoveAll("/usr/local/share/rhetor/" + fs.StudentFSIdentifier); err != nil {
		fmt.Printf("could not remove directory: %v\n", err)
	}
	// remove student .tgz from disk
	os.Remove(fileName)
	fmt.Println(result.Location)
	return nil
}
