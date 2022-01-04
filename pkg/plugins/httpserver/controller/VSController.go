package controller

import (
	"encoding/json"
	"net/http"
	"rpc-gateway/pkg/plugins/httpserver/service"
	"rpc-gateway/pkg/plugins/httpserver/util"
	"rpc-gateway/pkg/plugins/pool/grpc"
	"rpc-gateway/pkg/plugins/vs/kvs"
	"strings"
	"time"

	logging "rpc-gateway/pkg/core/log"

	"github.com/gin-gonic/gin"
)

// controller struct
type VSController struct {
	apiVersion string
	Service    *service.VSService
}

// asr default env params
var defaultAsrEnvParams = kvs.AsrEngineEnvParams{
	USER_ID:                      "1000",
	GROUP_ID:                     "1000",
	GRPC_LISTEN_IP:               "0.0.0.0",
	GRPC_LISTEN_PORT:             "31502",
	ASR_PLUGIN_ZHUIYI_ASR_SGR_ID: "sgr_v1.0.0",
	USE_PLUGIN:                   "zhuiyi",
	USE_PLUGIN_ALIAS:             "zhuiyi",
	AUDIO_FRAGMENT_TIME_MS:       "200",  // 200 ms
	TIMEOUT_SECOND:               "30",   // 30 s
	WAIT_RESULT_SECOND:           "5",    // 5 s
	ENABLE_VAD:                   "true", // vad 是否启用
	SAMPLE_RATE:                  "8000", // 采样率

	ENABLE_HOTWORD:                    "true",  // 是否开启热词
	ENABLE_DUMP:                       "false", // 是否保存音频
	ENABLE_PUNCTUATION_PREDICTION:     "false", // 是否标点拼音
	ENABLE_INVERSE_TEXT_NORMALIZATION: "false", // 是否开启数字转小写
	ASR_PLUGIN_ZHUIYI_MAX_SESSION:     "10",    // ASR 启用的最大路数
	ASR_LANGUAGE_MODEL_DOWNLOAD_URL:   "",      // 语言模型下载地址
	ASR_ACOUSTIC_MODEL_DOWNLOAD_URL:   "",      // 声学模型下载地址

	MYSQL_PORT:     "3306", // DB 设置
	MYSQL_IP:       "172.16.40.8",
	MYSQL_USER:     "root",
	MYSQL_PWD:      "uWXf87plmQGz8zMM",
	MYSQL_DATABASE: "zhuiyi",
}

// asr vad env
var defaultAsrVadEnvParams = kvs.AsrVadEnvParams{
	MODEL_DIR:        "model/vad_model/vad_model_v4/wavenet",
	SPEECH_TIME_THRE: "0.22", // 语音时间阈值
	SIL_TIME_THRE:    "0.55", // 静音时间阈值
	SPEECH_WEIGHT:    "0.65", // 语音概率权重
	SIL_WEIGHT:       "0.55", // 静音概率权重
	SPEECH_WIN_LEN:   "0.20", // 语音滑窗步长
	SIL_WIN_LEN:      "0.20",
}

// asr hotwords
var defaultAsrHotWordsEnvParams = kvs.AsrHotWordsEnvParams{
	HOT_WORDS: "",
}

// tts default env params
var defaultTtsEnvParams = kvs.TtsEngineEnvParams{
	USER_ID:                          "1010",
	GROUP_ID:                         "1010",
	GRPC_LISTEN_IP:                   "0.0.0.0",
	GRPC_LISTEN_PORT:                 "20800",
	VALUE:                            "default",
	USE_PLUGIN:                       "zhuiyi",
	USE_PLUGIN_ALIAS:                 "zhuiyi",
	TTS_PLUGIN_ZHUIYI_VOICE_ID:       "xiaona_16k,xiaona",
	ENABLE_DUMP:                      "false",
	ENABLE_CACHE:                     "true",
	TTS_PLUGIN_ZHUIYI_MAX_SESSION:    "10",
	TTS_PLUGIN_ZHUIYI_ENABLE_RT_MODE: "true",
	TTS_PLUGIN_ZHUIYI_ENABLE_CACHE:   "true",
	MYSQL_PORT:                       "3306", // DB 设置
	MYSQL_IP:                         "172.16.40.8",
	MYSQL_USER:                       "root",
	MYSQL_PWD:                        "uWXf87plmQGz8zMM",
	MYSQL_DATABASE:                   "zhuiyi",
}

// tts default word pinyin params
var defaultTtsWordPinyinParams = kvs.TtsEngineWrodPinyinParams{
	WORD_PINYIN: "",
}

// tts default word pinyin params
var defaultTtsFamilyNameParams = kvs.TtsEngineFamilyNameParams{
	FAMILY_NAME: "",
}

// get controller
func (vs *VSController) getCtl() *VSController {
	var svc *service.VSService
	return &VSController{"v1", svc}
}

// User godoc
// @Summary calc User api
// @Description get all users
// @Tags users
// @Accept json
// @Produce json
// @Param apiQuery query string false "used for api"
// @Success 200 {integer} string "{"code":0, "data":[{User},{User}], "message":"ok"}"
// @Failure 404 {string} string "method not found"
// @Failure 500 {string} string "Service Unavailable"
// @Router /users [get]
func (vs *VSController) CreateEngine(c *gin.Context) {
	// engine num
	sceneCode, existScene := c.GetPostForm("sceneCode")
	if !existScene {
		// send message
		util.SendError(c, "scene code not be null!")
		return
	}
	// engine concurrent num
	engineCCNum, existECCNum := c.GetPostForm("engineCCNum")
	if !existECCNum {
		// send message
		util.SendError(c, "engine concurrent num not be null!")
		return
	}
	// engine type
	engineType, existEtype := c.GetPostForm("engineType")
	if !existEtype {
		// send message
		util.SendError(c, "engine type not be null!")
		return
	}

	if strings.ToLower(engineType) == "asr" {
		// engine concurrent num
		defaultAsrEnvParams.ASR_PLUGIN_ZHUIYI_MAX_SESSION = engineCCNum
		// hotwords
		asrHotWords, existAsrHotWords := c.GetPostForm("asrHotWords")
		if existAsrHotWords {
			defaultAsrHotWordsEnvParams.HOT_WORDS = asrHotWords
		}
		// asr language model download url
		asrLanguageModelDownloadUrl, existAsrLangModelDownloadUrl := c.GetPostForm("asrLanguageModelDownloadUrl")
		if existAsrLangModelDownloadUrl {
			defaultAsrEnvParams.ASR_LANGUAGE_MODEL_DOWNLOAD_URL = asrLanguageModelDownloadUrl
		}
		// asr acoustic model download url
		asrAcousticModelDownloadUrl, existAsrAcousticModelDownloadUrl := c.GetPostForm("asrAcousticModelDownloadUrl")
		if existAsrAcousticModelDownloadUrl {
			defaultAsrEnvParams.ASR_ACOUSTIC_MODEL_DOWNLOAD_URL = asrAcousticModelDownloadUrl
		}
	} else if strings.ToLower(engineType) == "tts" {
		// tts voice id
		ttsVoiceId, existTtsVoiceId := c.GetPostForm("ttsVoiceId")
		if existTtsVoiceId {
			defaultTtsEnvParams.TTS_PLUGIN_ZHUIYI_VOICE_ID = ttsVoiceId
		}
		// word pinyin
		ttsBusiDict, existTtsBusiDict := c.GetPostForm("ttsBusiDict")
		if existTtsBusiDict {
			defaultTtsWordPinyinParams.WORD_PINYIN = ttsBusiDict
		}
		// family name
		ttsNameDict, existTtsNameDict := c.GetPostForm("ttsNameDict")
		if existTtsNameDict {
			defaultTtsFamilyNameParams.FAMILY_NAME = ttsNameDict
		}
		// engine concurrent num
		defaultTtsEnvParams.TTS_PLUGIN_ZHUIYI_MAX_SESSION = engineCCNum
	}

	// setting env params
	envParamsInf := make(map[string]interface{})
	engineEnvParams, existAsrEngineEnvParams := c.GetPostForm("busiConfigInfo")
	engineEnvParamsMap := make(map[string]interface{})
	if existAsrEngineEnvParams {
		if err := json.Unmarshal([]byte(engineEnvParams), &engineEnvParamsMap); err != nil {
			logging.Log.Error(err.Error())
			c.JSON(http.StatusOK, gin.H{
				"code":    -1,
				"message": err.Error(),
			})
			return
		}
		// asr envs
		if strings.ToLower(engineType) == "asr" {
			// vad
			// speech time thre
			if _, ok := engineEnvParamsMap["speechTimeThre"]; ok {
				defaultAsrVadEnvParams.SPEECH_TIME_THRE = engineEnvParamsMap["speechTimeThre"].(string)
			}
			// sil time thre
			if _, ok := engineEnvParamsMap["silTimeThre"]; ok {
				defaultAsrVadEnvParams.SIL_TIME_THRE = engineEnvParamsMap["silTimeThre"].(string)
			}
			// speech weight
			if _, ok := engineEnvParamsMap["speechWeight"]; ok {
				defaultAsrVadEnvParams.SPEECH_WEIGHT = engineEnvParamsMap["speechWeight"].(string)
			}
			// sil weight
			if _, ok := engineEnvParamsMap["silWeight"]; ok {
				defaultAsrVadEnvParams.SIL_WEIGHT = engineEnvParamsMap["silWeight"].(string)
			}
			// speech win len
			if _, ok := engineEnvParamsMap["speechWinLen"]; ok {
				defaultAsrVadEnvParams.SPEECH_WIN_LEN = engineEnvParamsMap["speechWinLen"].(string)
			}
			// business config
			if _, ok := engineEnvParamsMap["enableDump"]; ok {
				defaultAsrEnvParams.ENABLE_DUMP = engineEnvParamsMap["enableDump"].(string)
			}

			if _, ok := engineEnvParamsMap["enableHotword"]; ok {
				defaultAsrEnvParams.ENABLE_HOTWORD = engineEnvParamsMap["enableHotword"].(string)
			}
			// enable punctuation prediction
			if _, ok := engineEnvParamsMap["enablePunctuationPrediction"]; ok {
				defaultAsrEnvParams.ENABLE_PUNCTUATION_PREDICTION = engineEnvParamsMap["enablePunctuationPrediction"].(string)
			}
			// enable inverse text normalization
			if _, ok := engineEnvParamsMap["enableInverseTextNormalization"]; ok {
				defaultAsrEnvParams.ENABLE_INVERSE_TEXT_NORMALIZATION = engineEnvParamsMap["enableInverseTextNormalization"].(string)
			}
			envParamsInf["envParams"] = defaultAsrEnvParams
			envParamsInf["asrVadParams"] = defaultAsrVadEnvParams
			envParamsInf["asrHotWordsParams"] = defaultAsrHotWordsEnvParams
		} else if strings.ToLower(engineType) == "tts" {
			// tts envs
			// enable dump
			if _, ok := engineEnvParamsMap["enableDump"]; ok {
				defaultTtsEnvParams.ENABLE_DUMP = engineEnvParamsMap["enableDump"].(string)
			}
			// enable cache
			if _, ok := engineEnvParamsMap["enableCache"]; ok {
				defaultTtsEnvParams.ENABLE_CACHE = engineEnvParamsMap["enableCache"].(string)
				defaultTtsEnvParams.TTS_PLUGIN_ZHUIYI_ENABLE_CACHE = engineEnvParamsMap["enableCache"].(string)
			}
			envParamsInf["ttsWordPinyinParams"] = defaultTtsWordPinyinParams
			envParamsInf["ttsFamilyNameParams"] = defaultTtsFamilyNameParams
			envParamsInf["envParams"] = defaultTtsEnvParams
		}
	} else {
		if strings.ToLower(engineType) == "asr" {
			envParamsInf["asrVadParams"] = defaultAsrVadEnvParams
			envParamsInf["asrHotWordsParams"] = defaultAsrHotWordsEnvParams
			envParamsInf["envParams"] = defaultAsrEnvParams
		} else if strings.ToLower(engineType) == "tts" {
			envParamsInf["ttsWordPinyinParams"] = defaultTtsWordPinyinParams
			envParamsInf["ttsFamilyNameParams"] = defaultTtsFamilyNameParams
			envParamsInf["envParams"] = defaultTtsEnvParams
		}
	}
	// setting engine name
	serverName := strings.ToLower(engineType) + "-" + engineCCNum + "-" + sceneCode
	// create engine
	err := vs.getCtl().Service.CreateEngine(
		map[string]interface{}{
			"sceneCode":       sceneCode,
			"engineName":      sceneCode,
			"engineCCNum":     engineCCNum,
			"engineType":      strings.ToLower(engineType),
			"serverName":      serverName,
			"engineEnvParams": envParamsInf,
		},
	)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}
	/*
		// create grpc pool
		ecNum, errEcNum := strconv.Atoi(engineCCNum)
		if errEcNum != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    -1,
				"message": errEcNum,
			})
			return
		}
		// init pool by omp
		grpc.InitGrpcPoolByOmp(map[string]interface{}{
			"sceneCode":   sceneCode,
			"engineName":  sceneCode,
			"engineCCNum": ecNum,
			"engineType":  strings.ToLower(engineType),
			"serverHost":  serverName,
			"serverName":  serverName,
		})
	*/
	// send message
	util.SendMessage(c, util.Message{
		Code:    0,
		Message: "OK",
		Data:    "",
	})
}

// update engine
func (vs *VSController) UpdateEngine(c *gin.Context) {
	// engine num
	sceneCode, existScene := c.GetPostForm("sceneCode")
	if !existScene {
		// send message
		util.SendError(c, "scene code not be null!")
		return
	}
	// engine concurrent num
	engineCCNum, existECCNum := c.GetPostForm("engineCCNum")
	if !existECCNum {
		// send message
		util.SendError(c, "engine concurrent num not be null!")
		return
	}
	// engine type
	engineType, existEtype := c.GetPostForm("engineType")
	if !existEtype {
		// send message
		util.SendError(c, "engine type not be null!")
		return
	}

	if strings.ToLower(engineType) == "asr" {
		// engine concurrent num
		defaultAsrEnvParams.ASR_PLUGIN_ZHUIYI_MAX_SESSION = engineCCNum
		// hotwords
		asrHotWords, existAsrHotWords := c.GetPostForm("asrHotWords")
		if existAsrHotWords {
			defaultAsrHotWordsEnvParams.HOT_WORDS = asrHotWords
		}
		// asr language model download url
		asrLanguageModelDownloadUrl, existAsrLangModelDownloadUrl := c.GetPostForm("asrLanguageModelDownloadUrl")
		if existAsrLangModelDownloadUrl {
			defaultAsrEnvParams.ASR_LANGUAGE_MODEL_DOWNLOAD_URL = asrLanguageModelDownloadUrl
		}
		// asr acoustic model download url
		asrAcousticModelDownloadUrl, existAsrAcousticModelDownloadUrl := c.GetPostForm("asrAcousticModelDownloadUrl")
		if existAsrAcousticModelDownloadUrl {
			defaultAsrEnvParams.ASR_ACOUSTIC_MODEL_DOWNLOAD_URL = asrAcousticModelDownloadUrl
		}
	} else if strings.ToLower(engineType) == "tts" {
		// tts voice id
		ttsVoiceId, existTtsVoiceId := c.GetPostForm("ttsVoiceId")
		if existTtsVoiceId {
			defaultTtsEnvParams.TTS_PLUGIN_ZHUIYI_VOICE_ID = ttsVoiceId
		}
		// word pinyin
		ttsBusiDict, existTtsBusiDict := c.GetPostForm("ttsBusiDict")
		if existTtsBusiDict {
			defaultTtsWordPinyinParams.WORD_PINYIN = ttsBusiDict
		}
		// family name
		ttsNameDict, existTtsNameDict := c.GetPostForm("ttsNameDict")
		if existTtsNameDict {
			defaultTtsFamilyNameParams.FAMILY_NAME = ttsNameDict
		}
		// engine concurrent num
		defaultTtsEnvParams.TTS_PLUGIN_ZHUIYI_MAX_SESSION = engineCCNum
	}
	// setting env params
	envParamsInf := make(map[string]interface{})
	engineEnvParams, existAsrEngineEnvParams := c.GetPostForm("busiConfigInfo")
	engineEnvParamsMap := make(map[string]interface{})
	if existAsrEngineEnvParams {
		if err := json.Unmarshal([]byte(engineEnvParams), &engineEnvParamsMap); err != nil {
			logging.Log.Error(err.Error())
			c.JSON(http.StatusOK, gin.H{
				"code":    -1,
				"message": err.Error(),
			})
			return
		}
		// asr envs
		if strings.ToLower(engineType) == "asr" {
			// vad
			// speech time thre
			if _, ok := engineEnvParamsMap["speechTimeThre"]; ok {
				defaultAsrVadEnvParams.SPEECH_TIME_THRE = engineEnvParamsMap["speechTimeThre"].(string)
			}
			// sil time thre
			if _, ok := engineEnvParamsMap["silTimeThre"]; ok {
				defaultAsrVadEnvParams.SIL_TIME_THRE = engineEnvParamsMap["silTimeThre"].(string)
			}
			// speech weight
			if _, ok := engineEnvParamsMap["speechWeight"]; ok {
				defaultAsrVadEnvParams.SPEECH_WEIGHT = engineEnvParamsMap["speechWeight"].(string)
			}
			// sil weight
			if _, ok := engineEnvParamsMap["silWeight"]; ok {
				defaultAsrVadEnvParams.SIL_WEIGHT = engineEnvParamsMap["silWeight"].(string)
			}
			// speech win len
			if _, ok := engineEnvParamsMap["speechWinLen"]; ok {
				defaultAsrVadEnvParams.SPEECH_WIN_LEN = engineEnvParamsMap["speechWinLen"].(string)
			}
			// business config
			defaultAsrEnvParams.ENABLE_DUMP = engineEnvParamsMap["enableDump"].(string)
			if _, ok := engineEnvParamsMap["enableHotword"]; ok {
				defaultAsrEnvParams.ENABLE_HOTWORD = engineEnvParamsMap["enableHotword"].(string)
			}
			// enable punctuation prediction
			if _, ok := engineEnvParamsMap["enablePunctuationPrediction"]; ok {
				defaultAsrEnvParams.ENABLE_PUNCTUATION_PREDICTION = engineEnvParamsMap["enablePunctuationPrediction"].(string)
			}
			// enable inverse text normalization
			if _, ok := engineEnvParamsMap["enableInverseTextNormalization"]; ok {
				defaultAsrEnvParams.ENABLE_INVERSE_TEXT_NORMALIZATION = engineEnvParamsMap["enableInverseTextNormalization"].(string)
			}
			envParamsInf["envParams"] = defaultAsrEnvParams
			envParamsInf["asrVadParams"] = defaultAsrVadEnvParams
			envParamsInf["asrHotWordsParams"] = defaultAsrHotWordsEnvParams
		} else if strings.ToLower(engineType) == "tts" {
			// tts envs
			// enable dump
			if _, ok := engineEnvParamsMap["enableDump"]; ok {
				defaultTtsEnvParams.ENABLE_DUMP = engineEnvParamsMap["enableDump"].(string)
			}
			// enable cache
			if _, ok := engineEnvParamsMap["enableCache"]; ok {
				defaultTtsEnvParams.ENABLE_CACHE = engineEnvParamsMap["enableCache"].(string)
				defaultTtsEnvParams.TTS_PLUGIN_ZHUIYI_ENABLE_CACHE = engineEnvParamsMap["enableCache"].(string)
			}
			envParamsInf["ttsWordPinyinParams"] = defaultTtsWordPinyinParams
			envParamsInf["ttsFamilyNameParams"] = defaultTtsFamilyNameParams
			envParamsInf["envParams"] = defaultTtsEnvParams
		}
	} else {
		if strings.ToLower(engineType) == "asr" {
			envParamsInf["asrVadParams"] = defaultAsrVadEnvParams
			envParamsInf["asrHotWordsParams"] = defaultAsrHotWordsEnvParams
			envParamsInf["envParams"] = defaultAsrEnvParams
		} else if strings.ToLower(engineType) == "tts" {
			envParamsInf["ttsWordPinyinParams"] = defaultTtsWordPinyinParams
			envParamsInf["ttsFamilyNameParams"] = defaultTtsFamilyNameParams
			envParamsInf["envParams"] = defaultTtsEnvParams
		}
	}

	// setting engine name
	serverName := strings.ToLower(engineType) + "-" + engineCCNum + "-" + sceneCode
	// update engine
	err := vs.getCtl().Service.UpdateEngine(map[string]interface{}{
		"sceneCode":       sceneCode,
		"engineName":      sceneCode,
		"engineCCNum":     engineCCNum,
		"engineType":      strings.ToLower(engineType),
		"serverName":      serverName,
		"engineEnvParams": envParamsInf,
	})

	poolName := sceneCode
	go func() {
		for {
			time.Sleep(5 * time.Second)
			err := vs.getCtl().Service.CheckEngine(map[string]interface{}{
				"engineName":  sceneCode,
				"engineCCNum": engineCCNum,
				"engineType":  strings.ToLower(engineType),
				"serverName":  serverName,
			})
			if err == nil {
				logging.Log.Info("update engine ...")
				grpc.ReleaseGrpcPool(poolName, engineType)
				grpc.InitPoolFromExistEngineForUpdate(map[string]interface{}{
					"engineName":  sceneCode,
					"engineCCNum": engineCCNum,
					"engineType":  strings.ToLower(engineType),
					"serverName":  serverName,
				})
				return
			}
		}
	}()

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}
	// send message
	util.SendMessage(c, util.Message{
		Code:    0,
		Message: "OK",
		Data:    "",
	})
}

// delete engine
func (vs *VSController) DeleteEngine(c *gin.Context) {
	// engine num
	engineCCNum, existEngineCCNum := c.GetPostForm("engineCCNum")
	if !existEngineCCNum {
		// send message
		util.SendError(c, "scene code not be null!")
		return
	}

	sceneCode, existScene := c.GetPostForm("sceneCode")
	if !existScene {
		// send message
		util.SendError(c, "scene code not be null!")
		return
	}

	engineType, existEtype := c.GetPostForm("engineType")
	if !existEtype {
		// send message
		util.SendError(c, "engine type not be null!")
		return
	}
	// delete engine
	serverName := strings.ToLower(engineType) + "-" + engineCCNum + "-" + sceneCode
	err := vs.getCtl().Service.DeleteEngine(map[string]interface{}{
		"engineName": sceneCode,
		"serverName": serverName,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}
	// release pool
	poolName := sceneCode
	grpc.ReleaseGrpcPool(poolName, engineType)
	// send message
	util.SendMessage(c, util.Message{
		Code:    0,
		Message: "OK",
		Data:    "",
	})
}

// check engine
func (vs *VSController) CheckEngine(c *gin.Context) {
	// engine num
	engineCCNum, existEngineCCNum := c.GetPostForm("engineCCNum")
	if !existEngineCCNum {
		// send message
		util.SendError(c, "scene code not be null!")
		return
	}

	sceneCode, existScene := c.GetPostForm("sceneCode")
	if !existScene {
		// send message
		util.SendError(c, "scene code not be null!")
		return
	}

	engineType, existEtype := c.GetPostForm("engineType")
	if !existEtype {
		// send message
		util.SendError(c, "engine type not be null!")
		return
	}

	serverName := strings.ToLower(engineType) + "-" + engineCCNum + "-" + sceneCode
	err := vs.getCtl().Service.CheckEngine(map[string]interface{}{
		"engineName":  sceneCode,
		"engineCCNum": engineCCNum,
		"engineType":  strings.ToLower(engineType),
		"serverName":  serverName,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	util.SendMessage(c, util.Message{
		Code:    0,
		Message: "OK",
		Data:    "engine is ready",
	})
}
