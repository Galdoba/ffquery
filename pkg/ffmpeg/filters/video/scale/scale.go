package scale

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/utils"
)

// Scale представляет видеофильтр масштабирования (scale) из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#scale
type Scale struct {
	// --- Options specific to scale filter (39.221.1) ---

	// Width, W - Set the output video dimension expression. Default value is the input dimension.
	// If 0, input width is used. If -n (n>=1), maintains aspect ratio divisible by n.
	Width string `json:"width,omitempty"`
	// Height, H - Set the output video dimension expression. Default value is the input dimension.
	// If 0, input height is used. If -n (n>=1), maintains aspect ratio divisible by n.
	Height string `json:"height,omitempty"`
	// Size, S - Set the video size. For syntax see ffmpeg-utils manual.
	Size string `json:"size,omitempty"`
	// Eval - Specify when to evaluate width and height expressions.
	// Values: "init" (default), "frame".
	Eval string `json:"eval,omitempty"`
	// Interl - Set the interlacing mode. Values: 1 (force interlaced), 0 (no interlaced), -1 (auto).
	// Default 0.
	Interl *int `json:"interl,omitempty"`
	// Flags - Set libswscale scaling flags. See ffmpeg-scaler manual for complete list.
	Flags string `json:"flags,omitempty"`
	// Param0 - Set libswscale input parameter for scaling algorithms that need them.
	Param0 *float64 `json:"param0,omitempty"`
	// Param1 - Set libswscale input parameter for scaling algorithms that need them.
	Param1 *float64 `json:"param1,omitempty"`
	// Intent - Set the ICC rendering intent for color space transformation.
	// Values: "perceptual", "relative_colorimetric" (default), "absolute_colorimetric", "saturation".
	Intent string `json:"intent,omitempty"`

	// --- Options inherited from libswscale scaler ---

	// Scaler - Choose the scaling algorithm to use. Default "auto".
	// Values: "auto", "bilinear", "bicubic", "point"/"neighbor", "area", "gaussian",
	// "sinc", "lanczos", "spline".
	Scaler string `json:"scaler,omitempty"`
	// ScalerSub - Choose the scaling algorithm for chroma subsampling. Default "auto".
	// Same values as Scaler.
	ScalerSub string `json:"scaler_sub,omitempty"`
	// SwsFlags - Set the scaler flags. Deprecated in favor of Scaler. Default "bicubic".
	// Accepts many values (see documentation).
	SwsFlags string `json:"sws_flags,omitempty"`
	// SwsDither - Set the dithering algorithm. Default "auto".
	// Values: "auto", "none", "bayer", "ed", "a_dither", "x_dither".
	SwsDither string `json:"sws_dither,omitempty"`
	// Alphablend - Set alpha blending when input has alpha but output does not. Default "none".
	// Values: "uniform_color", "checkerboard", "none".
	Alphablend string `json:"alphablend,omitempty"`
	// SwsBackends - Set allowed swscale backends (flags). Default "auto".
	// Multiple backends may be combined with "+".
	// Values: "auto", "stable", "unstable", "all", "legacy", "c", "memcpy", "x86", "aarch64", "spirv".
	SwsBackends string `json:"sws_backends,omitempty"`

	// --- Color space options ---

	// InColorMatrix - Set input YCbCr color space type.
	// Values: "auto", "bt709", "fcc", "bt601", "bt470", "smpte170m", "smpte240m", "bt2020".
	InColorMatrix string `json:"in_color_matrix,omitempty"`
	// OutColorMatrix - Set output YCbCr color space type. Same values as InColorMatrix.
	OutColorMatrix string `json:"out_color_matrix,omitempty"`

	// InRange - Set input YCbCr sample range.
	// Values: "auto"/"unknown", "jpeg"/"full"/"pc", "mpeg"/"limited"/"tv".
	InRange string `json:"in_range,omitempty"`
	// OutRange - Set output YCbCr sample range. Same values as InRange.
	OutRange string `json:"out_range,omitempty"`

	// InChromaLoc - Set input chroma sample location.
	// Values: "auto"/"unknown", "left", "center", "topleft", "top", "bottomleft", "bottom".
	InChromaLoc string `json:"in_chroma_loc,omitempty"`
	// OutChromaLoc - Set output chroma sample location. Same values as InChromaLoc.
	OutChromaLoc string `json:"out_chroma_loc,omitempty"`

	// InPrimaries - Set input RGB primaries.
	// Values: "auto", "bt709", "bt470m", "bt470bg", "smpte170m", "smpte240m",
	// "film", "bt2020", "smpte428", "smpte431", "smpte432", "jedec-p22", "ebu3213".
	InPrimaries string `json:"in_primaries,omitempty"`
	// OutPrimaries - Set output RGB primaries. Same values as InPrimaries.
	OutPrimaries string `json:"out_primaries,omitempty"`

	// InTransfer - Set input transfer response curve (TRC).
	// Values: "auto", "bt709", "bt470m", "gamma22", "bt470bg", "gamma28",
	// "smpte170m", "smpte240m", "linear", "iec61966-2-1", "srgb", "iec61966-2-4",
	// "xvycc", "bt1361e", "bt2020-10", "bt2020-12", "smpte2084", "smpte428", "arib-std-b67".
	InTransfer string `json:"in_transfer,omitempty"`
	// OutTransfer - Set output transfer response curve. Same values as InTransfer.
	OutTransfer string `json:"out_transfer,omitempty"`

	// ForceOriginalAspectRatio - Enable decreasing/increasing output dimensions to keep original aspect ratio.
	// Values: "disable", "decrease", "increase".
	ForceOriginalAspectRatio string `json:"force_original_aspect_ratio,omitempty"`
	// ForceDivisibleBy - Ensure output dimensions are divisible by given integer.
	// Used with ForceOriginalAspectRatio.
	ForceDivisibleBy *int `json:"force_divisible_by,omitempty"`
	// ResetSAR - If true, output SAR is reset to 1. Default false.
	ResetSAR *bool `json:"reset_sar,omitempty"`

	// --- API-only options (not used in command line, included for completeness) ---

	// SrcW (API only) - Set source width. Not used in command line.
	SrcW int `json:"-"`
	// SrcH (API only) - Set source height. Not used in command line.
	SrcH int `json:"-"`
	// DstW (API only) - Set destination width. Not used in command line.
	DstW int `json:"-"`
	// DstH (API only) - Set destination height. Not used in command line.
	DstH int `json:"-"`
	// SrcFormat (API only) - Set source pixel format. Not used in command line.
	SrcFormat int `json:"-"`
	// DstFormat (API only) - Set destination pixel format. Not used in command line.
	DstFormat int `json:"-"`

	// SrcRange (boolean) - If 1, indicates source is full range. Default 0 (limited range).
	SrcRange *bool `json:"src_range,omitempty"`
	// DstRange (boolean) - If 1, enable full range for destination. Default 0 (limited range).
	DstRange *bool `json:"dst_range,omitempty"`
	// Gamma (boolean) - If 1, enable gamma correct scaling. Default 0.
	Gamma *bool `json:"gamma,omitempty"`

	err error
}

// New создаёт фильтр Scale с дефолтными значениями из документации и применяет переданные опции.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Scale {
	s := &Scale{}
	for _, opt := range opts {
		if err := s.apply(opt); err != nil {
			s.err = err
			break
		}
	}
	if s.err == nil {
		s.err = s.Validate()
	}
	return s
}

// apply – приватный метод, потребляет одну опцию, валидирует её и устанавливает соответствующее поле.
func (s *Scale) apply(opt options.Option) error {
	switch opt.Key {
	// width / w
	case "width", "w":
		if opt.Value == "" {
			return fmt.Errorf("scale: width expression cannot be empty")
		}
		s.Width = opt.Value

	// height / h
	case "height", "h":
		if opt.Value == "" {
			return fmt.Errorf("scale: height expression cannot be empty")
		}
		s.Height = opt.Value

	// size / s
	case "size", "s":
		if opt.Value == "" {
			return fmt.Errorf("scale: size cannot be empty")
		}
		s.Size = opt.Value

	// eval
	case "eval":
		if opt.Value != "init" && opt.Value != "frame" {
			return fmt.Errorf("scale: eval must be 'init' or 'frame', got %q", opt.Value)
		}
		s.Eval = opt.Value

	// interl
	case "interl":
		v, err := strconv.Atoi(opt.Value)
		if err != nil || v < -1 || v > 1 {
			return fmt.Errorf("scale: interl must be -1, 0, or 1, got %q", opt.Value)
		}
		s.Interl = &v

	// flags
	case "flags":
		if opt.Value == "" {
			return fmt.Errorf("scale: flags cannot be empty")
		}
		s.Flags = opt.Value

	// param0
	case "param0":
		p, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("scale: invalid param0 value %q", opt.Value)
		}
		s.Param0 = &p

	// param1
	case "param1":
		p, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("scale: invalid param1 value %q", opt.Value)
		}
		s.Param1 = &p

	// intent
	case "intent":
		if !isValidIntent(opt.Value) {
			return fmt.Errorf("scale: invalid intent %q", opt.Value)
		}
		s.Intent = opt.Value

	// scaler
	case "scaler":
		if !isValidScaler(opt.Value) {
			return fmt.Errorf("scale: invalid scaler %q", opt.Value)
		}
		s.Scaler = opt.Value

	// scaler_sub
	case "scaler_sub":
		if !isValidScaler(opt.Value) {
			return fmt.Errorf("scale: invalid scaler_sub %q", opt.Value)
		}
		s.ScalerSub = opt.Value

	// sws_flags
	case "sws_flags":
		if !isValidSwsFlags(opt.Value) {
			return fmt.Errorf("scale: invalid sws_flags %q", opt.Value)
		}
		s.SwsFlags = opt.Value

	// sws_dither
	case "sws_dither":
		if !isValidSwsDither(opt.Value) {
			return fmt.Errorf("scale: invalid sws_dither %q", opt.Value)
		}
		s.SwsDither = opt.Value

	// alphablend
	case "alphablend":
		if !isValidAlphablend(opt.Value) {
			return fmt.Errorf("scale: invalid alphablend %q", opt.Value)
		}
		s.Alphablend = opt.Value

	// sws_backends
	case "sws_backends":
		// валидация не проводится, т.к. это flags с комбинацией
		s.SwsBackends = opt.Value

	// in_color_matrix
	case "in_color_matrix":
		if !isValidColorMatrix(opt.Value) {
			return fmt.Errorf("scale: invalid in_color_matrix %q", opt.Value)
		}
		s.InColorMatrix = opt.Value

	// out_color_matrix
	case "out_color_matrix":
		if !isValidColorMatrix(opt.Value) {
			return fmt.Errorf("scale: invalid out_color_matrix %q", opt.Value)
		}
		s.OutColorMatrix = opt.Value

	// in_range
	case "in_range":
		if !isValidRange(opt.Value) {
			return fmt.Errorf("scale: invalid in_range %q", opt.Value)
		}
		s.InRange = opt.Value

	// out_range
	case "out_range":
		if !isValidRange(opt.Value) {
			return fmt.Errorf("scale: invalid out_range %q", opt.Value)
		}
		s.OutRange = opt.Value

	// in_chroma_loc
	case "in_chroma_loc":
		if !isValidChromaLoc(opt.Value) {
			return fmt.Errorf("scale: invalid in_chroma_loc %q", opt.Value)
		}
		s.InChromaLoc = opt.Value

	// out_chroma_loc
	case "out_chroma_loc":
		if !isValidChromaLoc(opt.Value) {
			return fmt.Errorf("scale: invalid out_chroma_loc %q", opt.Value)
		}
		s.OutChromaLoc = opt.Value

	// in_primaries
	case "in_primaries":
		if !isValidPrimaries(opt.Value) {
			return fmt.Errorf("scale: invalid in_primaries %q", opt.Value)
		}
		s.InPrimaries = opt.Value

	// out_primaries
	case "out_primaries":
		if !isValidPrimaries(opt.Value) {
			return fmt.Errorf("scale: invalid out_primaries %q", opt.Value)
		}
		s.OutPrimaries = opt.Value

	// in_transfer
	case "in_transfer":
		if !isValidTransfer(opt.Value) {
			return fmt.Errorf("scale: invalid in_transfer %q", opt.Value)
		}
		s.InTransfer = opt.Value

	// out_transfer
	case "out_transfer":
		if !isValidTransfer(opt.Value) {
			return fmt.Errorf("scale: invalid out_transfer %q", opt.Value)
		}
		s.OutTransfer = opt.Value

	// force_original_aspect_ratio
	case "force_original_aspect_ratio":
		if !isValidForceOriginalAspectRatio(opt.Value) {
			return fmt.Errorf("scale: invalid force_original_aspect_ratio %q", opt.Value)
		}
		s.ForceOriginalAspectRatio = opt.Value

	// force_divisible_by
	case "force_divisible_by":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n <= 0 {
			return fmt.Errorf("scale: force_divisible_by must be a positive integer, got %q", opt.Value)
		}
		s.ForceDivisibleBy = &n

	// reset_sar
	case "reset_sar":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("scale: reset_sar must be boolean, got %q", opt.Value)
		}
		s.ResetSAR = &b

	// src_range
	case "src_range":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("scale: src_range must be boolean, got %q", opt.Value)
		}
		s.SrcRange = &b

	// dst_range
	case "dst_range":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("scale: dst_range must be boolean, got %q", opt.Value)
		}
		s.DstRange = &b

	// gamma
	case "gamma":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("scale: gamma must be boolean, got %q", opt.Value)
		}
		s.Gamma = &b

	default:
		return fmt.Errorf("scale: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
func (s *Scale) Validate() error {
	if s.Size != "" && (s.Width != "" || s.Height != "") {
		return fmt.Errorf("scale: size cannot be used together with width or height")
	}
	// Если задан force_divisible_by, должен быть задан force_original_aspect_ratio
	if s.ForceDivisibleBy != nil && s.ForceOriginalAspectRatio == "" {
		return fmt.Errorf("scale: force_divisible_by requires force_original_aspect_ratio")
	}
	return nil
}

// String возвращает строку фильтра (например, "scale=w=iw/2:h=ih").
// При ошибке возвращает пустую строку.
func (s *Scale) String() string {
	if s.Err() != nil {
		return ""
	}
	var parts []string

	// width/height или size
	if s.Size != "" {
		parts = append(parts, "size="+s.Size)
	} else {
		if s.Width != "" {
			parts = append(parts, "w="+s.Width)
		}
		if s.Height != "" {
			parts = append(parts, "h="+s.Height)
		}
	}

	// Остальные опции добавляются только если явно заданы
	if s.Eval != "" {
		parts = append(parts, "eval="+s.Eval)
	}
	if s.Interl != nil {
		parts = append(parts, fmt.Sprintf("interl=%d", *s.Interl))
	}
	if s.Flags != "" {
		parts = append(parts, "flags="+s.Flags)
	}
	if s.Param0 != nil {
		parts = append(parts, fmt.Sprintf("param0=%v", *s.Param0))
	}
	if s.Param1 != nil {
		parts = append(parts, fmt.Sprintf("param1=%v", *s.Param1))
	}
	if s.Intent != "" {
		parts = append(parts, "intent="+s.Intent)
	}
	if s.Scaler != "" {
		parts = append(parts, "scaler="+s.Scaler)
	}
	if s.ScalerSub != "" {
		parts = append(parts, "scaler_sub="+s.ScalerSub)
	}
	if s.SwsFlags != "" {
		parts = append(parts, "sws_flags="+s.SwsFlags)
	}
	if s.SwsDither != "" {
		parts = append(parts, "sws_dither="+s.SwsDither)
	}
	if s.Alphablend != "" {
		parts = append(parts, "alphablend="+s.Alphablend)
	}
	if s.SwsBackends != "" {
		parts = append(parts, "sws_backends="+s.SwsBackends)
	}
	if s.InColorMatrix != "" {
		parts = append(parts, "in_color_matrix="+s.InColorMatrix)
	}
	if s.OutColorMatrix != "" {
		parts = append(parts, "out_color_matrix="+s.OutColorMatrix)
	}
	if s.InRange != "" {
		parts = append(parts, "in_range="+s.InRange)
	}
	if s.OutRange != "" {
		parts = append(parts, "out_range="+s.OutRange)
	}
	if s.InChromaLoc != "" {
		parts = append(parts, "in_chroma_loc="+s.InChromaLoc)
	}
	if s.OutChromaLoc != "" {
		parts = append(parts, "out_chroma_loc="+s.OutChromaLoc)
	}
	if s.InPrimaries != "" {
		parts = append(parts, "in_primaries="+s.InPrimaries)
	}
	if s.OutPrimaries != "" {
		parts = append(parts, "out_primaries="+s.OutPrimaries)
	}
	if s.InTransfer != "" {
		parts = append(parts, "in_transfer="+s.InTransfer)
	}
	if s.OutTransfer != "" {
		parts = append(parts, "out_transfer="+s.OutTransfer)
	}
	if s.ForceOriginalAspectRatio != "" {
		parts = append(parts, "force_original_aspect_ratio="+s.ForceOriginalAspectRatio)
	}
	if s.ForceDivisibleBy != nil {
		parts = append(parts, fmt.Sprintf("force_divisible_by=%d", *s.ForceDivisibleBy))
	}
	if s.ResetSAR != nil {
		if *s.ResetSAR {
			parts = append(parts, "reset_sar=1")
		} else {
			parts = append(parts, "reset_sar=0")
		}
	}
	if s.SrcRange != nil {
		if *s.SrcRange {
			parts = append(parts, "src_range=1")
		} else {
			parts = append(parts, "src_range=0")
		}
	}
	if s.DstRange != nil {
		if *s.DstRange {
			parts = append(parts, "dst_range=1")
		} else {
			parts = append(parts, "dst_range=0")
		}
	}
	if s.Gamma != nil {
		if *s.Gamma {
			parts = append(parts, "gamma=1")
		} else {
			parts = append(parts, "gamma=0")
		}
	}

	if len(parts) == 0 {
		return "scale"
	}
	return "scale=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (s *Scale) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (s *Scale) ProvideOption() options.Option {
	if s.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: s.String()}
}

// --- Вспомогательные функции валидации ---

// func parseBool(value string) (bool, error) {
// 	switch strings.ToLower(value) {
// 	case "1", "true", "yes", "on":
// 		return true, nil
// 	case "0", "false", "no", "off":
// 		return false, nil
// 	default:
// 		return false, fmt.Errorf("invalid boolean value %q", value)
// 	}
// }

func isValidIntent(v string) bool {
	switch v {
	case "perceptual", "relative_colorimetric", "absolute_colorimetric", "saturation":
		return true
	}
	return false
}

func isValidScaler(v string) bool {
	switch v {
	case "auto", "bilinear", "bicubic", "point", "neighbor", "area", "gaussian", "sinc", "lanczos", "spline":
		return true
	}
	return false
}

func isValidSwsFlags(v string) bool {
	// Принимаем любую строку, но проверяем, что она содержит только допустимые флаги?
	// Для простоты не строго проверяем, т.к. флаги могут комбинироваться.
	// Валидируем, что v не пустая и не содержит пробелов (это флаги через + или одиночные)
	if v == "" {
		return false
	}
	// допустимые значения
	validFlags := map[string]bool{
		"fast_bilinear": true, "bilinear": true, "bicubic": true, "experimental": true,
		"neighbor": true, "area": true, "bicublin": true, "gauss": true, "sinc": true,
		"lanczos": true, "spline": true, "print_info": true, "accurate_rnd": true,
		"full_chroma_int": true, "full_chroma_inp": true, "bitexact": true, "unstable": true,
	}
	// Проверяем каждый флаг, если они разделены '+'
	for flag := range strings.SplitSeq(v, "+") {
		if !validFlags[flag] {
			return false
		}
	}
	return true
}

func isValidSwsDither(v string) bool {
	switch v {
	case "auto", "none", "bayer", "ed", "a_dither", "x_dither":
		return true
	}
	return false
}

func isValidAlphablend(v string) bool {
	switch v {
	case "uniform_color", "checkerboard", "none":
		return true
	}
	return false
}

func isValidColorMatrix(v string) bool {
	switch v {
	case "auto", "bt709", "fcc", "bt601", "bt470", "smpte170m", "smpte240m", "bt2020":
		return true
	}
	return false
}

func isValidRange(v string) bool {
	switch v {
	case "auto", "unknown", "jpeg", "full", "pc", "mpeg", "limited", "tv":
		return true
	}
	return false
}

func isValidChromaLoc(v string) bool {
	switch v {
	case "auto", "unknown", "left", "center", "topleft", "top", "bottomleft", "bottom":
		return true
	}
	return false
}

func isValidPrimaries(v string) bool {
	switch v {
	case "auto", "bt709", "bt470m", "bt470bg", "smpte170m", "smpte240m", "film",
		"bt2020", "smpte428", "smpte431", "smpte432", "jedec-p22", "ebu3213":
		return true
	}
	return false
}

func isValidTransfer(v string) bool {
	switch v {
	case "auto", "bt709", "bt470m", "gamma22", "bt470bg", "gamma28", "smpte170m",
		"smpte240m", "linear", "iec61966-2-1", "srgb", "iec61966-2-4", "xvycc",
		"bt1361e", "bt2020-10", "bt2020-12", "smpte2084", "smpte428", "arib-std-b67":
		return true
	}
	return false
}

func isValidForceOriginalAspectRatio(v string) bool {
	switch v {
	case "disable", "decrease", "increase":
		return true
	}
	return false
}

// Так как String() скрывает ошибку, для отладки и JSON-сценариев удобно иметь метод, возвращающий ошибку явно.
func (s *Scale) FilterString() (string, error) {
	if err := s.Err(); err != nil {
		return "", err
	}
	return s.String(), nil
}
