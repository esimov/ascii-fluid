package detector

import (
	"errors"

	pigo "github.com/esimov/pigo/core"
)

// perturbFact represents the perturbation factor used for pupils/eyes localization
const perturbFact = 63

var (
	cascade          []byte
	puplocCascade    []byte
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	imgParams        *pigo.ImageParams
	err              error
)

// UnpackCascades unpack all of used cascade files.
func (d *Detector) UnpackCascades() error {
	p := pigo.NewPigo()

	cascade, err = d.ParseCascade("/cascade/facefinder")
	if err != nil {
		return errors.New("error reading the facefinder cascade file")
	}
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	faceClassifier, err = p.Unpack(cascade)
	if err != nil {
		return errors.New("error unpacking the facefinder cascade file")
	}

	plc := pigo.NewPuplocCascade()

	puplocCascade, err = d.ParseCascade("/cascade/puploc")
	if err != nil {
		return errors.New("error reading the puploc cascade file")
	}

	puplocClassifier, err = plc.UnpackCascade(puplocCascade)
	if err != nil {
		return errors.New("error unpacking the puploc cascade file")
	}
	return nil
}

// DetectFaces runs the cluster detection over the webcam frame
// received as a pixel array and returns the detected faces.
func (d *Detector) DetectFaces(pixels []uint8, width, height int) [][]int {
	results := d.clusterDetection(pixels, width, height)
	dets := make([][]int, len(results))

	for i := 0; i < len(results); i++ {
		dets[i] = append(dets[i], results[i].Row, results[i].Col, results[i].Scale, int(results[i].Q))
	}
	return dets
}

// DetectLeftPupil detects the left pupil
func (d *Detector) DetectLeftPupil(results []int) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      results[0] - int(0.085*float32(results[2])),
		Col:      results[1] - int(0.185*float32(results[2])),
		Scale:    float32(results[2]) * 0.4,
		Perturbs: 63,
	}
	leftEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
	if leftEye.Row > 0 && leftEye.Col > 0 {
		return leftEye
	}
	return nil
}

// DetectRightPupil detects the right pupil
func (d *Detector) DetectRightPupil(results []int) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      results[0] - int(0.085*float32(results[2])),
		Col:      results[1] + int(0.185*float32(results[2])),
		Scale:    float32(results[2]) * 0.4,
		Perturbs: perturbFact,
	}
	rightEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
	if rightEye.Row > 0 && rightEye.Col > 0 {
		return rightEye
	}
	return nil
}

// clusterDetection runs Pigo face detector core methods
// and returns a cluster with the detected faces coordinates.
func (d *Detector) clusterDetection(pixels []uint8, width, height int) []pigo.Detection {
	imgParams = &pigo.ImageParams{
		Pixels: pixels,
		Rows:   width,
		Cols:   height,
		Dim:    height,
	}
	cParams := pigo.CascadeParams{
		MinSize:     100,
		MaxSize:     1200,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = faceClassifier.ClusterDetections(dets, 0.1)

	return dets
}
