package vector3

import "math"

// V represents a 3D vector
type V struct {
	X float64
	Y float64
	Z float64
}

// Add returns the sum of two vectors
func (v V) Add(other V) V {
	return V{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

// Subtract returns the difference between two vectors
func (v V) Subtract(other V) V {
	return V{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

// Scale multiplies all components of the vector by a scalar
func (v V) Scale(scalar float64) V {
	return V{
		X: v.X * scalar,
		Y: v.Y * scalar,
		Z: v.Z * scalar,
	}
}

// Cross returns the cross product of two vectors
func (v V) Cross(other V) V {
	return V{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// Dot returns the dot product of two vectors
func (v V) Dot(other V) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Normalize returns the normalized vector
func (v V) Normalize() V {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return V{
		X: v.X / length,
		Y: v.Y / length,
		Z: v.Z / length,
	}
}
