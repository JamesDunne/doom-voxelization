package matrix4

import (
	"awesomeProject/vector3"
	"math"
)

// M represents a 4x4 matrix
type M [16]float64

// Multiply returns the product of two matrices
func (m M) Multiply(other M) M {
	result := M{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				result[i*4+j] += m[i*4+k] * other[k*4+j]
			}
		}
	}
	return result
}

func (m M) Translate(vec vector3.V) M {
	m[12] += vec.X
	m[13] += vec.Y
	m[14] += vec.Z
	return m
}

// Transform applies the matrix transformation to a vector
func (m M) Transform(v vector3.V) vector3.V {
	x := m[0]*v.X + m[1]*v.Y + m[2]*v.Z + m[3]
	y := m[4]*v.X + m[5]*v.Y + m[6]*v.Z + m[7]
	z := m[8]*v.X + m[9]*v.Y + m[10]*v.Z + m[11]
	return vector3.V{X: x, Y: y, Z: z}
}

// RotationMatrix returns a rotation matrix that rotates around the x, y, and z axes
// by the specified angles (in radians), in that order.
func RotationMatrix(xAngle, yAngle, zAngle float64) M {
	cosX, sinX := math.Cos(xAngle), math.Sin(xAngle)
	cosY, sinY := math.Cos(yAngle), math.Sin(yAngle)
	cosZ, sinZ := math.Cos(zAngle), math.Sin(zAngle)

	// Create the rotation matrices for each axis
	xRot := M{
		1, 0, 0, 0,
		0, cosX, -sinX, 0,
		0, sinX, cosX, 0,
		0, 0, 0, 1,
	}
	yRot := M{
		cosY, 0, sinY, 0,
		0, 1, 0, 0,
		-sinY, 0, cosY, 0,
		0, 0, 0, 1,
	}
	zRot := M{
		cosZ, -sinZ, 0, 0,
		sinZ, cosZ, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}

	// Combine the rotation matrices into a single matrix
	return zRot.Multiply(yRot).Multiply(xRot)
}

// RotationMatrixInlined returns a rotation matrix that rotates around the x, y, and z axes
// by the specified angles (in radians), in that order.
// TODO verify against RotationMatrix
func RotationMatrixInlined(xAngle, yAngle, zAngle float64) M {
	cosX, sinX := math.Cos(xAngle), math.Sin(xAngle)
	cosY, sinY := math.Cos(yAngle), math.Sin(yAngle)
	cosZ, sinZ := math.Cos(zAngle), math.Sin(zAngle)

	// Calculate the elements of the final rotation matrix
	a := cosY * cosZ
	b := -cosY * sinZ
	c := sinY
	d := sinX*sinY*cosZ + cosX*sinZ
	e := -sinX*sinY*sinZ + cosX*cosZ
	f := -sinX * cosY
	g := -cosX*sinY*cosZ + sinX*sinZ
	h := cosX*sinY*sinZ + sinX*cosZ
	i := cosX * cosY

	// Return the final rotation matrix
	return M{
		a, d, g, 0,
		b, e, h, 0,
		c, f, i, 0,
		0, 0, 0, 1,
	}
}

func RotationX(xAngle float64) M {
	sinX, cosX := math.Sincos(xAngle)

	xRot := M{
		1, 0, 0, 0,
		0, cosX, -sinX, 0,
		0, sinX, cosX, 0,
		0, 0, 0, 1,
	}

	return xRot
}

func RotationY(yAngle float64) M {
	sinY, cosY := math.Sincos(yAngle)

	yRot := M{
		cosY, 0, sinY, 0,
		0, 1, 0, 0,
		-sinY, 0, cosY, 0,
		0, 0, 0, 1,
	}

	return yRot
}

func RotationZ(zAngle float64) M {
	sinZ, cosZ := math.Sincos(zAngle)

	zRot := M{
		cosZ, -sinZ, 0, 0,
		sinZ, cosZ, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}

	return zRot
}
