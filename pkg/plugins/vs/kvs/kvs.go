package kvs

import (
	"encoding/json"
	"errors"
	"fmt"
	logging "rpc-gateway/pkg/core/log"
	"strings"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var vsViper *viper.Viper
var kvsEnabled bool = true
var apiKvsClient *K8sClient
var namespace, manifestFilePath, minCpuNum, minMemoryMum string

const (
	POD_READY_STATUS_READY      = "Ready"
	POD_READY_STATUS_READY_TRUE = "True"
	POD_STATUS_RUNNING          = "Running"
)

var (
	ErrKvsNotEnabled     = errors.New("vs is not support!")
	ErrEngineExists      = errors.New("engine pod exists!")
	ErrEngineStatusCheck = errors.New("engine pod is not ready!")
	ErrEngineNotFound    = errors.New("please make sure whether engine exists!")
)

// asr engine envs
type AsrEngineEnvParams struct {
	// UsePlugin           string `json:"usePlugin"`
	// AudioFragmentTimeMs string `json:"audioFragmentTimeMs"`
	// TimeoutSecond       string `json:"timeoutSecond"`
	// WaitResultSecond    string `json:"waitResultSecond"`
	// EnableVad           string `json:"enableVad"`
	// SampleRate          string `json:"sampleRate"`

	// SpeechTimeThre string `json:"speechTimeThre"`
	// SilTimeThre    string `json:"silTimeThre"`
	// SpeechWeight   string `json:"speechWeight"`
	// SilWeight      string `json:"silWeight"`
	// SpeechWinLen   string `json:"speechWinLen"`

	// EnableHotword                  string `json:"enableHotword"`
	// EnableDump                     string `json:"enableDump"`
	// EnablePunctuationPrediction    string `json:"enablePunctuationPrediction"`
	// EnableInverseTextNormalization string `json:"enableInverseTextNormalization"`

	// MySqlPort     string `json:"mySqlPort"`
	// MySqlIp       string `json:"mySqlIp"`
	// MySqlUser     string `json:"mySqlUser"`
	// MySqlPwd      string `json:"mySqlPwd"`
	// MysqlDatabase string `json:"mysqlDatabase"`

	// default env
	USER_ID                      string
	GROUP_ID                     string
	GRPC_LISTEN_IP               string
	GRPC_LISTEN_PORT             string
	VALUE                        string
	ASR_PLUGIN_ZHUIYI_ASR_SGR_ID string
	// config env
	USE_PLUGIN                        string
	USE_PLUGIN_ALIAS                  string
	AUDIO_FRAGMENT_TIME_MS            string
	TIMEOUT_SECOND                    string
	WAIT_RESULT_SECOND                string
	ENABLE_VAD                        string
	SAMPLE_RATE                       string
	ENABLE_HOTWORD                    string
	ENABLE_DUMP                       string
	ENABLE_PUNCTUATION_PREDICTION     string
	ENABLE_INVERSE_TEXT_NORMALIZATION string
	ASR_PLUGIN_ZHUIYI_MAX_SESSION     string
	ASR_LANGUAGE_MODEL_DOWNLOAD_URL   string
	ASR_ACOUSTIC_MODEL_DOWNLOAD_URL   string

	// db env
	MYSQL_PORT     string
	MYSQL_IP       string
	MYSQL_USER     string
	MYSQL_PWD      string
	MYSQL_DATABASE string
}

// asr vad env parmas
type AsrVadEnvParams struct {
	MODEL_DIR        string `json:"model_dir"`
	SPEECH_TIME_THRE string `json:"speech_time_thre"`
	SIL_TIME_THRE    string `json:"sil_time_thre"`
	SPEECH_WEIGHT    string `json:"speech_weight"`
	SIL_WEIGHT       string `json:"sil_weight"`
	SPEECH_WIN_LEN   string `json:"speech_win_len"`
	SIL_WIN_LEN      string `json:"sil_win_len"`
}

// asr hot word
type AsrHotWordsEnvParams struct {
	HOT_WORDS string
}

// tts engine envs
type TtsEngineEnvParams struct {
	// default env
	USER_ID                          string
	GROUP_ID                         string
	GRPC_LISTEN_IP                   string
	GRPC_LISTEN_PORT                 string
	VALUE                            string
	USE_PLUGIN                       string
	USE_PLUGIN_ALIAS                 string
	TTS_PLUGIN_ZHUIYI_VOICE_ID       string
	ENABLE_DUMP                      string
	ENABLE_CACHE                     string
	TTS_PLUGIN_ZHUIYI_MAX_SESSION    string
	TTS_PLUGIN_ZHUIYI_ENABLE_RT_MODE string
	TTS_PLUGIN_ZHUIYI_ENABLE_CACHE   string
	// db env
	MYSQL_PORT     string
	MYSQL_IP       string
	MYSQL_USER     string
	MYSQL_PWD      string
	MYSQL_DATABASE string
}

// tts engine word pinyin
type TtsEngineWrodPinyinParams struct {
	WORD_PINYIN string
}

// tts engine family name
type TtsEngineFamilyNameParams struct {
	FAMILY_NAME string
}

// init
func init() {
	vsViper = viper.New()
	vsViper.SetConfigName("VSConfig")
	vsViper.SetConfigType("yaml")
	vsViper.AddConfigPath("config/")
	err := vsViper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error get pool config file: %s", err))
	}

	// get config
	vsRoot := vsViper.AllSettings()["vs"]
	if vsRoot == nil {
		return
	}

	kvsSetting := vsRoot.(map[string]interface{})["kvs"].(map[string]interface{})
	// whether enabled kvs
	kvsEnabled = kvsSetting[strings.ToLower("ENABLED")].(bool)
	if !kvsEnabled {
		return
	}
	namespace = kvsSetting[strings.ToLower("NAMESPACE")].(string)
	manifestFilePath = kvsSetting[strings.ToLower("DEPLOY_FILE_PATH")].(string)
	minCpuNum = kvsSetting[strings.ToLower("MIN_CPU")].(string)
	minMemoryMum = kvsSetting[strings.ToLower("MIN_MEMORY")].(string)

	config, errConfig := rest.InClusterConfig()
	if errConfig != nil {
		logging.Log.Error(errConfig.Error())
		return
	}
	// creates the clientset
	clientset, errClientSet := kubernetes.NewForConfig(config)
	if errClientSet != nil {
		panic(errClientSet.Error())
	}
	// get api client
	apiKvsClient = &K8sClient{
		Clientset: clientset,
		Namespace: namespace,
	}
}

// get engines
func GetEngineSvcs(data map[string]string) *corev1.ServiceList {
	return apiKvsClient.GetServicesByLabel(data)
}

// get engines
func GetEnginePods(data map[string]string) *corev1.PodList {
	return apiKvsClient.GetPodsByLabel(data)
}

// create engine
func CreateEngine(data map[string]interface{}) error {
	engineType := data["engineType"].(string)
	// setting pod
	// setting name
	deployName := data["serverName"].(string)
	// setting engine name
	engineName := data["engineName"].(string)
	// setting cpu, memory num
	cpuNum := minCpuNum
	if _, exist := data["engineCCNum"]; exist {
		cpuNum = data["engineCCNum"].(string)
	}
	memoryNum := minMemoryMum
	if _, exist := data["engineMemory"]; exist {
		memoryNum = data["engineMemory"].(string)
	}
	// check pod if exists
	errCheckEngine := CheckEngine(map[string]interface{}{
		"engineName":  data["engineName"].(string),
		"engineCCNum": cpuNum,
	})
	// pod exists
	if errCheckEngine == nil {
		// return ErrEngineExists
		logging.Log.Info(ErrEngineExists)
	}
	logging.Log.Info("engine env params: ", data["engineEnvParams"])
	// env vars
	engineEnvParams := data["engineEnvParams"].(map[string]interface{})
	// setting manifest path
	var deployFilePath string
	// var engineEnvParams interface{}
	switch engineType {
	case "asr":
		deployFilePath = manifestFilePath + "asr/"
		// create configmap
		// create vad cm
		errCreateVadCm := createConfigMap(map[string]interface{}{
			"deployFile": deployFilePath + "vad-configmap.yaml",
			"deployName": deployName + "-vad-cm",
			"engineName": engineName,
			"engineType": engineType,
			"cpuNum":     cpuNum,
			"cmData":     engineEnvParams["asrVadParams"],
			"subPath":    "vad.yaml",
			"keyEnabled": true,
		})
		if errCreateVadCm != nil {
			return errCreateVadCm
		}
		// create hotwords cm
		errCreateHotWordsCm := createConfigMap(map[string]interface{}{
			"deployFile": deployFilePath + "hotwords-configmap.yaml",
			"deployName": deployName + "-hotwords-cm",
			"engineName": engineName,
			"engineType": engineType,
			"cpuNum":     cpuNum,
			"cmData":     engineEnvParams["asrHotWordsParams"],
			"subPath":    "hotwords.dict",
			"keyEnabled": false,
		})
		if errCreateHotWordsCm != nil {
			return errCreateHotWordsCm
		}
	case "tts":
		deployFilePath = manifestFilePath + "tts/"
		errCreateWordPinyinCm := createConfigMap(map[string]interface{}{
			"deployFile": deployFilePath + "wordpinyin-configmap.yaml",
			"deployName": deployName + "-wordpinyin-cm",
			"engineName": engineName,
			"engineType": engineType,
			"cpuNum":     cpuNum,
			"cmData":     engineEnvParams["ttsWordPinyinParams"],
			"subPath":    "cn_word2pinyin.dict",
			"keyEnabled": false,
		})
		if errCreateWordPinyinCm != nil {
			return errCreateWordPinyinCm
		}

		errCreateFamilyNameCm := createConfigMap(map[string]interface{}{
			"deployFile": deployFilePath + "familyname-configmap.yaml",
			"deployName": deployName + "-familyname-cm",
			"engineName": engineName,
			"engineType": engineType,
			"cpuNum":     cpuNum,
			"cmData":     engineEnvParams["ttsFamilyNameParams"],
			"subPath":    "familyname.txt",
			"keyEnabled": false,
		})
		if errCreateFamilyNameCm != nil {
			return errCreateFamilyNameCm
		}
	}
	// create pod
	errCreatePod := createPod(map[string]interface{}{
		"deployFile":      deployFilePath + "deployment.yaml",
		"deployName":      deployName,
		"cpuNum":          cpuNum,
		"memoryNum":       memoryNum,
		"engineName":      engineName,
		"engineType":      engineType,
		"engineEnvParams": engineEnvParams["envParams"],
	})

	if errCreatePod != nil {
		return errCreatePod
	}

	// create service
	errCreateSvc := createService(map[string]string{
		"deployFile": deployFilePath + "service.yaml",
		"deployName": deployName,
		"engineName": engineName,
		"engineType": engineType,
		"cpuNum":     cpuNum,
	})

	if errCreateSvc != nil {
		return errCreateSvc
	}

	return nil
}

// delete engine
func DeleteEngine(data map[string]interface{}) error {
	// setting pod
	// setting name
	deployName := data["serverName"].(string)
	// create pod
	errDeletePod := deletePod(deployName)
	if errDeletePod != nil {
		return errDeletePod
	}
	// create service
	errDeleteService := deleteService(deployName)
	if errDeleteService != nil {
		return errDeleteService
	}
	return nil
}

// update engine
func UpdateEngine(data map[string]interface{}) error {
	// get engine svcs
	svcs := GetEngineSvcs(map[string]string{
		"engineName": data["engineName"].(string),
	})
	// get engine pods
	pods := GetEnginePods(map[string]string{
		"engineName": data["engineName"].(string),
	})
	// update engine
	if len(svcs.Items) > 0 {
		engineData := strings.Split(svcs.Items[0].Name, "-")
		engineCCNum := engineData[1]
		if engineCCNum == data["engineCCNum"].(string) {
			CreateEngine(data)
		} else {
			// delete engine
			errDeleteEngine := DeleteEngine(map[string]interface{}{
				"serverName": svcs.Items[0].Name,
			})
			if errDeleteEngine != nil {
				return errDeleteEngine
			}
			// create engine
			errCreateEngine := CreateEngine(data)
			if errCreateEngine != nil {
				return errCreateEngine
			}
		}
	} else if len(pods.Items) > 0 {
		engineData := strings.Split(pods.Items[0].Name, "-")
		engineCCNum := engineData[1]
		if engineCCNum == data["engineCCNum"].(string) {
			CreateEngine(data)
		} else {
			// delete engine
			errDeleteEngine := DeleteEngine(map[string]interface{}{
				"serverName": pods.Items[0].Name,
			})
			if errDeleteEngine != nil {
				return errDeleteEngine
			}
			// create engine
			errCreateEngine := CreateEngine(data)
			if errCreateEngine != nil {
				return errCreateEngine
			}
		}
	} else {
		return ErrEngineNotFound
	}

	return nil
}

// check engine
func CheckEngine(data map[string]interface{}) error {
	// check pod
	sData := make(map[string]string)
	for k, v := range data {
		sData[k] = v.(string)
	}
	return checkPod(sData)
}

// create service
func createConfigMap(data map[string]interface{}) error {
	if !kvsEnabled {
		return ErrKvsNotEnabled
	}
	fileBytes := GetResourceYaml(data["deployFile"].(string))
	spec := apiKvsClient.UnmarshalConfigMap(fileBytes)
	spec.Name = data["deployName"].(string)
	spec.ObjectMeta.SetLabels(map[string]string{
		"engineName":  data["engineName"].(string),
		"engineType":  data["engineType"].(string),
		"engineCCNum": data["cpuNum"].(string),
		"engine":      "zhuiyi.ai." + data["engineType"].(string),
	})

	jsonEnvs, _ := json.Marshal(data["cmData"])
	envsMap := make(map[string]string)
	if err := json.Unmarshal([]byte(jsonEnvs), &envsMap); err != nil {
		return errors.New("pod envs to map failed ...")
	}
	// cm data
	var dataString string
	if data["keyEnabled"].(bool) {
		for k, v := range envsMap {
			dataString += k + ": " + v + "\n"
		}
	} else {
		for _, v := range envsMap {
			dataString += v + "\n"
		}
	}
	spec.Data = map[string]string{
		data["subPath"].(string): dataString,
	}
	// deploy cm
	deployErr := apiKvsClient.DeployConfigMap(spec)
	if deployErr != nil {
		logging.Log.Error("create service error", deployErr)
		return deployErr
	}
	return nil
}

// create pod
func createPod(data map[string]interface{}) error {
	if !kvsEnabled {
		return ErrKvsNotEnabled
	}
	fileBytes := GetResourceYaml(data["deployFile"].(string))
	spec := apiKvsClient.UnmarshalDeployment(fileBytes)
	spec.Name = data["deployName"].(string)
	for k, pod := range spec.Spec.Template.Spec.Containers {
		if pod.Name == "engine-server" {
			// setting envs
			envVars := []corev1.EnvVar{}
			jsonEnvs, _ := json.Marshal(data["engineEnvParams"])
			envsMap := make(map[string]string)
			if err := json.Unmarshal([]byte(jsonEnvs), &envsMap); err != nil {
				return errors.New("pod envs to map failed ...")
			}
			for k, v := range envsMap {
				envVars = append(envVars, corev1.EnvVar{
					Name:  k,
					Value: v,
				})
			}

			spec.Spec.Template.Spec.Containers[k].Env = envVars
			// setting resources
			spec.Spec.Template.Spec.Containers[k].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse(data["cpuNum"].(string)),
				corev1.ResourceMemory: resource.MustParse(data["memoryNum"].(string)),
			}
			spec.Spec.Template.Spec.Containers[k].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse(data["cpuNum"].(string)),
				corev1.ResourceMemory: resource.MustParse(data["memoryNum"].(string)),
			}
			// setting tts model
			if data["engineType"].(string) == "tts" {
				spec.Spec.Template.Spec.Volumes = append(spec.Spec.Template.Spec.Volumes, corev1.Volume{
					Name: "familyname-data",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: data["deployName"].(string) + "-familyname-cm",
							},
						},
					},
				})
			}
		} else if pod.Name == "tts-model" {
			spec.Spec.Template.Spec.Volumes = append(spec.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: "familyname-data",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: data["deployName"].(string) + "-familyname-cm",
						},
					},
				},
			})
		}
	}
	// setting selector
	newSelector := new(metav1.LabelSelector)
	newSelector.MatchLabels = make(map[string]string)
	newSelector.MatchLabels["engineName"] = data["engineName"].(string)
	newSelector.MatchLabels["engineType"] = data["engineType"].(string)
	spec.Spec.Selector = newSelector
	// setting labels
	spec.Spec.Template.ObjectMeta.Labels = map[string]string{
		"engineName":  data["engineName"].(string),
		"engineType":  data["engineType"].(string),
		"engineCCNum": data["cpuNum"].(string),
	}
	// setting volumes
	if data["engineType"].(string) == "asr" {
		// setting vad cm
		spec.Spec.Template.Spec.Volumes = append(spec.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "vad-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: data["deployName"].(string) + "-vad-cm",
					},
				},
			},
		})
		// setting hotwords cm
		spec.Spec.Template.Spec.Volumes = append(spec.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "hotwords-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: data["deployName"].(string) + "-hotwords-cm",
					},
				},
			},
		})
	} else if data["engineType"].(string) == "tts" {
		spec.Spec.Template.Spec.Volumes = append(spec.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "wordpinyin-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: data["deployName"].(string) + "-wordpinyin-cm",
					},
				},
			},
		})
	}

	// deploy deployment
	deployErr := apiKvsClient.DeployDeployment(spec)
	if deployErr != nil {
		logging.Log.Error("create pod error", deployErr)
		return deployErr
	}
	return nil
}

// create service
func createService(data map[string]string) error {
	if !kvsEnabled {
		return ErrKvsNotEnabled
	}
	fileBytes := GetResourceYaml(data["deployFile"])
	spec := apiKvsClient.UnmarshalService(fileBytes)
	spec.Name = data["deployName"]
	spec.Spec.Selector = map[string]string{
		"engineName": data["engineName"],
		"engineType": data["engineType"],
	}

	spec.ObjectMeta.SetLabels(map[string]string{
		"engineName":  data["engineName"],
		"engineType":  data["engineType"],
		"engineCCNum": data["cpuNum"],
		"engine":      "zhuiyi.ai." + data["engineType"],
	})
	deployErr := apiKvsClient.DeployService(spec)
	if deployErr != nil {
		logging.Log.Error("create service error", deployErr)
		return deployErr
	}
	return nil
}

// delete pod
func deletePod(deployName string) error {
	if !kvsEnabled {
		return ErrKvsNotEnabled
	}
	// delete pod
	return apiKvsClient.ResDelete("Deployment", deployName)
}

// delete service
func deleteService(deployName string) error {
	if !kvsEnabled {
		return ErrKvsNotEnabled
	}
	// delete service
	return apiKvsClient.ResDelete("Service", deployName)
}

// check pod
func checkPod(data map[string]string) error {
	if !kvsEnabled {
		return ErrKvsNotEnabled
	}
	// get svc list
	svcList := apiKvsClient.GetServicesByLabel(data)
	if len(svcList.Items) < 1 {
		return ErrEngineNotFound
	}
	// get pod list
	podList := apiKvsClient.GetPodsByLabel(data)
	if len(podList.Items) > 0 {
		for _, pod := range podList.Items {
			// podCheckStatus := false
			podStatus := string(pod.Status.Phase)
			for _, condition := range pod.Status.Conditions {
				if condition.Type == POD_READY_STATUS_READY {
					if condition.Status == POD_READY_STATUS_READY_TRUE && podStatus == POD_STATUS_RUNNING {
					} else {
						return ErrEngineStatusCheck
					}
				}
			}
		}
	} else {
		return ErrEngineNotFound
	}
	return nil
}

func GetEndpointsByName(name string) (*corev1.Endpoints, error) {
	endPoints, err := apiKvsClient.GetEndpointsByName(name)
	if err != nil {
		return nil, err
	}
	return endPoints, nil
}
