package ingest

import (
	"errors"
	"math"
	"time"
)

type LocationPoint struct {
	DriverID   string    `json:"driver_id"`
	OrderID    string    `json:"order_id,omitempty"`
	TS         time.Time `json:"ts"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	SpeedMS    float64   `json:"speed_m_s"`
	HeadingDeg float64   `json:"heading_deg"`
	AccuracyM  float64   `json:"accuracy_m"`
	Battery    float64   `json:"battery"`
	Source     string    `json:"source"`
	Motion     string    `json:"motion"`
	Seq        int64     `json:"seq"`
}

func (p *LocationPoint) Validate(maxAcc float64) error {
	if p.DriverID == "" {
		return errors.New("driver_id required")
	}
	if math.Abs(p.Lat) > 90 || math.Abs(p.Lng) > 180 {
		return errors.New("invalid lat/lng")
	}
	if p.AccuracyM <= 0 || p.AccuracyM > maxAcc {
		return errors.New("poor accuracy")
	}
	// timestamp default
	if p.TS.IsZero() {
		p.TS = time.Now().UTC()
	}
	return nil
}
