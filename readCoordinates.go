package readcoordinates

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

type latitude float64
type longtitude float64

var (
	PointWithBiggestLatitude   Point
	PointWithLeastLatitude     Point
	PointWithBiggestLongtitude Point
	PointWithLeastLongtitude   Point
)

const (
	minutesInDegree float64 = 60.
	secondsInDegree float64 = 3600.
	jpgRegexp               = `.*\.(jpg|JPG)$`
)

type Point struct {
	Longtitude longtitude
	Latitude   latitude
	Filename   string
}

func newPoint(Longtitude float64, Latitude float64, filename string) *Point {

	p := Point{Filename: filename}
	p.Longtitude = longtitude(Longtitude)
	p.Latitude = latitude(Latitude)
	return &p
}
func createPointArray(filenames []string, position map[string][2]float64) (points []Point) {
	for _, value := range filenames {
		NewPoint := newPoint(position[value][0], position[value][1], value)
		points = append(points, *NewPoint)
	}
	return

}
func createFilteredFolder(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}
}

func getFilenames(path string) []string {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	filenames := make([]string, 0)
	re := regexp.MustCompile(jpgRegexp)
	for _, file := range files {
		if re.MatchString(file.Name()) {
			filenames = append(filenames, file.Name())
		}
	}

	return filenames
}

func mapFilesToCoordinates(filenames []string, sourceFolder string, filteredFolder string) map[string][2]float64 {
	position := make(map[string][2]float64)
	for _, filename := range filenames {
		LatitudeAndLongtitude, err := retrievePosition(sourceFolder, filename, filteredFolder)
		if err != nil {
			moveToFilteredFolder(sourceFolder, filename, filteredFolder)
			continue
		}
		position[filename] = LatitudeAndLongtitude

	}

	return position
}

func decodeImg(file *os.File, path string, filename string, filteredFolder string) (*exif.Exif, error) {
	exifFile, err := exif.Decode(file)
	if err != nil {
		moveToFilteredFolder(path, filename, filteredFolder)
		return nil, err
	}

	return exifFile, nil
}

func getGPSLatitudeAndGPSLongtitude(file *os.File, path, filename string, filteredFolder string) (tiffLongitude *tiff.Tag, tiffLatitude *tiff.Tag, err error) {
	exifFile, err := decodeImg(file, path, filename, filteredFolder)
	if err != nil {
		return nil, nil, err
	}

	tiffLongitude, err = exifFile.Get(exif.GPSLongitude)
	if err != nil {
		moveToFilteredFolder(path, filename, filteredFolder)
		return nil, nil, err
	}
	tiffLatitude, err = exifFile.Get(exif.GPSLatitude)

	if err != nil {
		moveToFilteredFolder(path, filename, filteredFolder)
		return nil, nil, err
	}

	return
}

func retrievePosition(path, filename string, filteredFolder string) (position [2]float64, err error) {
	image, err := os.Open(path + "\\" + filename)

	if err != nil {
		return
	}
	defer image.Close()

	tiffLongitude, tiffLatitude, err := getGPSLatitudeAndGPSLongtitude(image, path, filename, filteredFolder)
	if err != nil {
		return
	}
	position[0] = fromArrayFloatToFloat64(fromStringToArrayFloat(tiffLongitude.String()))
	position[1] = fromArrayFloatToFloat64(fromStringToArrayFloat(tiffLatitude.String()))
	return
}

func moveToFilteredFolder(path string, filename string, filteredFolder string) {
	oldPlace := path + "\\" + filename
	newPlace := filteredFolder + "\\" + filename
	if err := os.Rename(oldPlace, newPlace); err != nil {
		log.Fatalf("failed to move file to filtered folder: %s", err.Error())
	}
}

func fromStringToArrayFloat(position string) (returning []float64) {
	temp := strings.Trim(position, "\"[]")
	substrings := strings.Split(temp, "\",\"")
	fmt.Println(substrings)
	for _, value := range substrings {
		temp := strings.Split(value, "/")
		divident, err := strconv.ParseFloat(temp[0], 64)
		if err != nil {
			log.Fatal(err)
		}
		divider, err := strconv.ParseFloat(temp[1], 64)
		if err != nil {
			log.Fatal(err)
		}
		returning = append(returning, divident/divider)

	}
	return
}
func fromArrayFloatToFloat64(coordinates []float64) (newCoordinates float64) {
	newCoordinates += coordinates[1]
	newCoordinates += coordinates[2] / minutesInDegree
	newCoordinates += coordinates[3] / secondsInDegree
	return
}
func findExtremums(points []Point) {
	i := 1
	for _, value := range points {
		if i == 1 {
			PointWithBiggestLatitude = value
			PointWithBiggestLongtitude = value
			PointWithLeastLatitude = value
			PointWithLeastLongtitude = value
			continue
		}
		if value.Longtitude < PointWithLeastLongtitude.Longtitude {
			PointWithLeastLongtitude = value
		}
		if value.Longtitude > PointWithBiggestLongtitude.Longtitude {
			PointWithBiggestLongtitude = value
		}
		if value.Latitude < PointWithLeastLatitude.Latitude {
			PointWithLeastLatitude = value
		}
		if value.Latitude > PointWithBiggestLatitude.Latitude {
			PointWithBiggestLatitude = value
		}
	}
}
func GetPoints(sourceFolder string, filteredFolder string) (coordinates []Point) {
	createFilteredFolder(filteredFolder)

	if sourceFolder == "" ||
		filteredFolder == "" {
		log.Fatal("ERROR: all flags must be set")
	}

	exif.RegisterParsers(mknote.All...)

	filenames := getFilenames(sourceFolder)

	filesToPosition := mapFilesToCoordinates(filenames, sourceFolder, filteredFolder)

	coordinates = createPointArray(filenames, filesToPosition)
	return
}
