package phase1

import (
	"bufio"
	"io"
	"math"
	"math/big"
	"math/bits"
	"os"
	"runtime"

	"github.com/bnbchain/zkbnb-setup/common"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/fft"
)

func butterflyG1(a *bn254.G1Affine, b *bn254.G1Affine) {
	t := *a
	a.Add(a, b)
	b.Sub(&t, b)
}

func butterflyG2(a *bn254.G2Affine, b *bn254.G2Affine) {
	t := *a
	a.Add(a, b)
	b.Sub(&t, b)
}

// kerDIF8 is a kernel that process a FFT of size 8
func kerDIF8G1(a []bn254.G1Affine, twiddles [][]fr.Element, stage int) {
	butterflyG1(&a[0], &a[4])
	butterflyG1(&a[1], &a[5])
	butterflyG1(&a[2], &a[6])
	butterflyG1(&a[3], &a[7])

	var twiddle big.Int
	twiddles[stage+0][1].BigInt(&twiddle)
	a[5].ScalarMultiplication(&a[5], &twiddle)
	twiddles[stage+0][2].BigInt(&twiddle)
	a[6].ScalarMultiplication(&a[6], &twiddle)
	twiddles[stage+0][3].BigInt(&twiddle)
	a[7].ScalarMultiplication(&a[7], &twiddle)
	butterflyG1(&a[0], &a[2])
	butterflyG1(&a[1], &a[3])
	butterflyG1(&a[4], &a[6])
	butterflyG1(&a[5], &a[7])
	twiddles[stage+1][1].BigInt(&twiddle)
	a[3].ScalarMultiplication(&a[3], &twiddle)
	twiddles[stage+1][1].BigInt(&twiddle)
	a[7].ScalarMultiplication(&a[7], &twiddle)
	butterflyG1(&a[0], &a[1])
	butterflyG1(&a[2], &a[3])
	butterflyG1(&a[4], &a[5])
	butterflyG1(&a[6], &a[7])
}

// kerDIF8 is a kernel that process a FFT of size 8
func kerDIF8G2(a []bn254.G2Affine, twiddles [][]fr.Element, stage int) {
	butterflyG2(&a[0], &a[4])
	butterflyG2(&a[1], &a[5])
	butterflyG2(&a[2], &a[6])
	butterflyG2(&a[3], &a[7])

	var twiddle big.Int
	twiddles[stage+0][1].BigInt(&twiddle)
	a[5].ScalarMultiplication(&a[5], &twiddle)
	twiddles[stage+0][2].BigInt(&twiddle)
	a[6].ScalarMultiplication(&a[6], &twiddle)
	twiddles[stage+0][3].BigInt(&twiddle)
	a[7].ScalarMultiplication(&a[7], &twiddle)
	butterflyG2(&a[0], &a[2])
	butterflyG2(&a[1], &a[3])
	butterflyG2(&a[4], &a[6])
	butterflyG2(&a[5], &a[7])
	twiddles[stage+1][1].BigInt(&twiddle)
	a[3].ScalarMultiplication(&a[3], &twiddle)
	twiddles[stage+1][1].BigInt(&twiddle)
	a[7].ScalarMultiplication(&a[7], &twiddle)
	butterflyG2(&a[0], &a[1])
	butterflyG2(&a[2], &a[3])
	butterflyG2(&a[4], &a[5])
	butterflyG2(&a[6], &a[7])
}

func difFFTG1(a []bn254.G1Affine, twiddles [][]fr.Element, stage, maxSplits int, chDone chan struct{}) {
	if chDone != nil {
		defer close(chDone)
	}

	n := len(a)
	if n == 1 {
		return
	} else if n == 8 {
		kerDIF8G1(a, twiddles, stage)
		return
	}
	m := n >> 1

	butterflyG1(&a[0], &a[m])

	var twiddle big.Int
	for i := 1; i < m; i++ {
		butterflyG1(&a[i], &a[i+m])
		twiddles[stage][i].BigInt(&twiddle)
		a[i+m].ScalarMultiplication(&a[i+m], &twiddle)
	}

	if m == 1 {
		return
	}

	nextStage := stage + 1
	if stage < maxSplits {
		chDone := make(chan struct{}, 1)
		go difFFTG1(a[m:n], twiddles, nextStage, maxSplits, chDone)
		difFFTG1(a[0:m], twiddles, nextStage, maxSplits, nil)
		<-chDone
	} else {
		difFFTG1(a[0:m], twiddles, nextStage, maxSplits, nil)
		difFFTG1(a[m:n], twiddles, nextStage, maxSplits, nil)
	}
}
func difFFTG2(a []bn254.G2Affine, twiddles [][]fr.Element, stage, maxSplits int, chDone chan struct{}) {
	if chDone != nil {
		defer close(chDone)
	}

	n := len(a)
	if n == 1 {
		return
	} else if n == 8 {
		kerDIF8G2(a, twiddles, stage)
		return
	}
	m := n >> 1

	butterflyG2(&a[0], &a[m])

	var twiddle big.Int
	for i := 1; i < m; i++ {
		butterflyG2(&a[i], &a[i+m])
		twiddles[stage][i].BigInt(&twiddle)
		a[i+m].ScalarMultiplication(&a[i+m], &twiddle)
	}

	if m == 1 {
		return
	}

	nextStage := stage + 1
	if stage < maxSplits {
		chDone := make(chan struct{}, 1)
		go difFFTG2(a[m:n], twiddles, nextStage, maxSplits, chDone)
		difFFTG2(a[0:m], twiddles, nextStage, maxSplits, nil)
		<-chDone
	} else {
		difFFTG2(a[0:m], twiddles, nextStage, maxSplits, nil)
		difFFTG2(a[m:n], twiddles, nextStage, maxSplits, nil)
	}
}

func lagrangifyG1(file *os.File, position int64, N int, domain *fft.Domain) error {
	// Seek to position
	if _, err := file.Seek(position, io.SeekStart); err != nil {
		return err
	}
	// Use buffered IO to write parameters efficiently
	buffSize := int(math.Pow(2, 20))
	reader := bufio.NewReaderSize(file, buffSize)
	writer := bufio.NewWriterSize(file, buffSize)
	defer writer.Flush()
	dec := bn254.NewDecoder(reader)
	enc := bn254.NewEncoder(writer)

	buff := make([]bn254.G1Affine, N)
	for i := 0; i < N; i++ {
		if err := dec.Decode(&buff[i]); err != nil {
			return err
		}
	}
	numCPU := uint64(runtime.NumCPU())
	maxSplits := bits.TrailingZeros64(ecc.NextPowerOfTwo(numCPU))
	difFFTG1(buff, domain.TwiddlesInv, 0, maxSplits, nil)
	common.BitReverseG1(buff)
	var invBigint big.Int
	domain.CardinalityInv.BigInt(&invBigint)
	common.Parallelize(len(buff), func(start, end int) {
		for i := start; i < end; i++ {
			buff[i].ScalarMultiplication(&buff[i], &invBigint)
		}
	})
	// Append to the end
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	// Serialize it
	if err := enc.Encode(buff); err != nil {
		return err
	}
	return nil
}

func lagrangifyG2(file *os.File, position int64, N int, domain *fft.Domain) error {
	// Seek to position
	if _, err := file.Seek(position, io.SeekStart); err != nil {
		return err
	}
	// Use buffered IO to write parameters efficiently
	buffSize := int(math.Pow(2, 20))
	reader := bufio.NewReaderSize(file, buffSize)
	writer := bufio.NewWriterSize(file, buffSize)
	defer writer.Flush()
	dec := bn254.NewDecoder(reader)
	enc := bn254.NewEncoder(writer)

	buff := make([]bn254.G2Affine, N)
	for i := 0; i < N; i++ {
		if err := dec.Decode(&buff[i]); err != nil {
			return err
		}
	}
	numCPU := uint64(runtime.NumCPU())
	maxSplits := bits.TrailingZeros64(ecc.NextPowerOfTwo(numCPU))
	difFFTG2(buff, domain.TwiddlesInv, 0, maxSplits, nil)
	common.BitReverseG2(buff)
	var invBigint big.Int
	domain.CardinalityInv.BigInt(&invBigint)
	common.Parallelize(len(buff), func(start, end int) {
		for i := start; i < end; i++ {
			buff[i].ScalarMultiplication(&buff[i], &invBigint)
		}
	})
	// Append to the end
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	// Serialize it
	if err := enc.Encode(buff); err != nil {
		return err
	}

	return nil
}
