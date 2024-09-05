package common

// TwoDimByteSlice2StringSlice: converts two-dimensional byte slices to string slices
// params:
// - byteSlice: two-dimensional byte slices
// return:
// - string slice
func TwoDimByteSlice2StringSlice(byteSlice [][]byte) []string {
	stringSlice := make([]string, len(byteSlice))
	for i := range byteSlice {
		stringSlice[i] = string(byteSlice[i])
	}
	return stringSlice
}

// TwoDimByteSlice2StringSlice: converts string slices to two-dimensional byte slices
// params:
// - strSlice: string slices
// return:
// - two-dimensional byte slices
func StringSlice2TwoDimByteSlice(strSlice []string) [][]byte {
	byteSlice := make([][]byte, len(strSlice))
	for i := range strSlice {
		byteSlice[i] = []byte(strSlice[i])
	}
	return byteSlice
}

// String2ByteSlice: converts string slices to two-dimensional byte slices
// params:
// - msg: string slices
// return:
// - two-dimensional byte slices
func String2ByteSlice(msg []string) [][]byte {
	res := make([][]byte, 0)
	if len(msg) == 0 {
		return res
	} else {
		for _, val := range msg {
			res = append(res, []byte(val))
		}
		return res
	}
}

// StringSlice2OneDimByteSlicee: converts string slices to one-dimensional byte slices
// params:
// - strSlice: string slices
// return:
// - one-dimensional byte slices
func StringSlice2OneDimByteSlice(strSlice []string) []byte {
	byteSlice := make([][]byte, len(strSlice))
	for i := range strSlice {
		byteSlice[i] = []byte(strSlice[i])
	}
	return TwoDimByteSlice2OneDimByteSlice(byteSlice)
}

// TwoDimByteSlice2OneDimByteSlic: converts two-dimensional byte slices to one-dimensional byte slices
// params:
// - twoDimByteSlice: two-dimensional byte slices
// return:
// - one-dimensional byte slices
func TwoDimByteSlice2OneDimByteSlice(twoDimByteSlice [][]byte) []byte {
	byteSlice := make([]byte, 0)
	for i := range twoDimByteSlice {
		byteSlice = append(byteSlice, twoDimByteSlice[i]...)
	}
	return byteSlice
}

// CutOffTwoDimByteSlice: the tail end of the two-dimensional section was excised
// params:
// - slice: slices to be processed
// - splitPoint: excision point for treatment
func CutOffTwoDimByteSlice(slice [][]byte, splitPoint int) [][]byte {
	if splitPoint == 0 {
		return make([][]byte, 0)
	}

	var resColumns [][]byte
	for _, row := range slice {
		if len(row) < splitPoint {
			resColumns = append(resColumns, row)
		} else {
			resColumns = append(resColumns, row[:splitPoint])
		}
	}

	return resColumns
}
