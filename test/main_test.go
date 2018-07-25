package test

import (
	"testing"
	"math/rand"
	"os"
	"fmt"
	"image"
	"image/color"
	"image/png"

)

func TestRand(t *testing.T) {
	r := rand.New(rand.NewSource(7814))
	k := r.Intn(1)

	for i := 0; i < 5; i++ {
		t.Log("rand 1 return ", k)
	}
}

func RandomCrop(src []byte, dst []byte, h int, w int, ch int, cw int, s int, r *rand.Rand) (ah int, aw int) {
	bw := w - cw
	bh := h - ch

	aw = r.Intn(bw)
	ah = r.Intn(bh)

	for y := ah; y < ah + ch; y++ {
		for x := aw; x < aw + cw; x++ {
			di := (y - ah) * cw * s + (x - aw) * s
			si := (y * w + x) * s

			dst[di] = src[si]
			dst[di + 1] = src[si + 1]
			dst[di + 2] = src[si + 2]
		}
	}

	return ah, aw
}


func TestRGB(t *testing.T) {
	r := rand.New(rand.NewSource(1478))
	//k := r.Intn(1)

	f, _ := os.Open("a.rgb")
	w := 896
	h := 448
	s := 3

	cw := 448
	ch := 224

	//bw := w - cw
	//bh := h - ch

	buf := make([]byte, w * s * h, w * s * h)
	n, _ := f.Read(buf[:cap(buf)])
	f.Close()

	t.Log("Full size SRC", w * s * h, len(buf), " read ", n)

	ob := make([]byte, cw * ch * s, cw * ch * s)
	for i := 0; i < 10; i++ {
		//aw := r.Intn(bw)
		//ah := r.Intn(bh)
		//
		//t.Log("random crop ", ah, aw)
		//for y := ah; y < ah + ch; y++ {
		//	for x := aw; x < aw + cw; x++ {
		//		di := (y - ah) * cw * s + (x - aw) * s
		//		si := (y * w + x) * s
		//
		//		//di := (x - aw) * ch * s + (y - ah) * s
		//		//si := (x * ch + y) * s
		//
		//		//t.Log("@idx ", di, " = ", si)
		//		//ob[di:di+3] = buf[si:si+3]
		//		//copy(ob[di:di+3], buf[si:si+3])
		//		//t.Log("test ", ob[di], buf[si])
		//		ob[di] = buf[si]
		//		ob[di + 1] = buf[si + 1]
		//		ob[di + 2] = buf[si + 2]
		//	}
		//}

		//main.RandomCrop()
		ah, aw := RandomCrop(buf, ob, h, w, ch, cw, s, r)

		ob = ob[:cw * ch * s]
		o1, _ := os.Create(fmt.Sprintf("%d_%d.rgb", ah, aw))
		o1.Write(ob[:cw * ch * s])
		defer o1.Close()

		m := image.NewRGBA(image.Rect(0, 0, cw, ch))
		for y := 0; y < ch; y++ {
			for x := 0; x < cw; x++ {
				idx := (y * cw  + x) * s
				//t.Log("IDX", idx)
				c := color.RGBA{
					ob[idx],
					ob[idx + 1],
					ob[idx + 2],
					255,
				}

				m.Set(x, y, c)
			}
		}
		o, _ := os.Create(fmt.Sprintf("%d_%d.png", ah, aw))
		defer o.Close()
		png.Encode(o, m)
	}
}

func UpdateBuf(src []byte, dst []byte)  {
	dst[0] = src[1]
}

func TestBuf(t *testing.T) {
	s := make([]byte, 4, 4)
	copy(s[:4], []byte("1234"))

	d := make([]byte, 2, 2)

	UpdateBuf(s, d)

	t.Log("d [0,1]", d[0], d[1])

}