package util

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	authApi "k8s.io/api/authorization/v1"
	coreApi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
)

const (
	QUERY_PARAMETER_USER_ID    = "userId"
	QUERY_PARAMETER_USER_GROUP = "userGroup"
)

var Clientset *kubernetes.Clientset
var config *restclient.Config
var AuditResourceList []string

func init() {
	// creates the in-cluster config
	var err error
	config, err = restclient.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	config.Burst = 100
	config.QPS = 100
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
}

func SetResponse(res http.ResponseWriter, outString string, outJson interface{}, status int) http.ResponseWriter {

	//set Cors
	// res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	res.Header().Set("Access-Control-Max-Age", "3628800")
	res.Header().Set("Access-Control-Expose-Headers", "Content-Type, X-Requested-With, Accept, Authorization, Referer, User-Agent")

	//set Out
	if outJson != nil {
		res.Header().Set("Content-Type", "application/json")
		js, err := json.Marshal(outJson)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		//set StatusCode
		res.WriteHeader(status)
		res.Write(js)
		return res

	} else {
		//set StatusCode
		res.WriteHeader(status)
		res.Write([]byte(outString))
		return res

	}
}

func UpdateAuditResourceList() {
	AuditResourceList = []string{"users"}
	tmp := make(map[string]struct{})
	fullName := make(map[string]struct{})
	apiGroupList := &metav1.APIGroupList{}
	data, err := Clientset.RESTClient().Get().AbsPath("/apis/").DoRaw(context.TODO())
	if err != nil {
		klog.Errorln(err)
	}
	if err := json.Unmarshal(data, apiGroupList); err != nil {
		klog.Errorln(err)
	}

	for _, apiGroup := range apiGroupList.Groups {
		for _, version := range apiGroup.Versions {
			apiResourceList := &metav1.APIResourceList{}
			path := strings.Replace("/apis/{GROUPVERSION}", "{GROUPVERSION}", version.GroupVersion, -1)
			data, err := Clientset.RESTClient().Get().AbsPath(path).DoRaw(context.TODO())
			if err != nil {
				klog.Errorln(err)
			}
			if err := json.Unmarshal(data, apiResourceList); err != nil {
				klog.Errorln(err)
			}

			for _, apiResource := range apiResourceList.APIResources {
				fullName[apiResource.Name] = struct{}{}
				if !strings.Contains(apiResource.Name, "/") {
					if _, ok := tmp[apiResource.Name]; !ok {
						tmp[apiResource.Name] = struct{}{}
					}
				}
			}
		}
	}

	//corev1 group
	apiResourceList := &metav1.APIResourceList{}
	data, err = Clientset.RESTClient().Get().AbsPath("/api/v1").DoRaw(context.TODO())
	if err != nil {
		klog.Errorln(err)
	}
	if err := json.Unmarshal(data, apiResourceList); err != nil {
		klog.Errorln(err)
	}
	for _, apiResource := range apiResourceList.APIResources {
		fullName[apiResource.Name] = struct{}{}
		if !strings.Contains(apiResource.Name, "/") {
			if _, ok := tmp[apiResource.Name]; !ok {
				tmp[apiResource.Name] = struct{}{}
			}
		}
	}

	// map to string

	for k, _ := range tmp {
		AuditResourceList = append(AuditResourceList, k)
	}

}

func CreateSubjectAccessReview(userId string, userGroups []string, group string, resource string, namespace string, name string, verb string) (*authApi.SubjectAccessReview, error) {
	sar := &authApi.SubjectAccessReview{
		Spec: authApi.SubjectAccessReviewSpec{
			ResourceAttributes: &authApi.ResourceAttributes{
				Group:     group,
				Resource:  resource,
				Namespace: namespace,
				Name:      name,
				Verb:      verb,
			},
			User:   userId,
			Groups: userGroups,
		},
	}

	sarResult, err := Clientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		klog.Errorln(err)
		return nil, err
	}

	return sarResult, nil
}

func GetAccessibleNS(userId string, labelSelector string, userGroups []string) coreApi.NamespaceList {
	var nsList = &coreApi.NamespaceList{}
	klog.Infoln("userId : ", userId)

	// // 1. Get UserGroup List if Exists
	// userDetail := getUserDetailWithoutToken(userId)
	// var userGroups []string
	// if userDetail["groups"] != nil {
	// 	for _, userGroup := range userDetail["groups"].([]interface{}) {
	// 		userGroups = append(userGroups, userGroup.(string))
	// 	}
	// }

	for _, userGroup := range userGroups {
		klog.Infoln("userGroupName : ", userGroup)
	}

	// 2. Check If User has NS List Role
	nsListRuleReview := authApi.SubjectAccessReview{
		Spec: authApi.SubjectAccessReviewSpec{
			ResourceAttributes: &authApi.ResourceAttributes{
				Resource: "namespaces",
				Verb:     "list",
				Group:    "",
			},
			User:   userId,
			Groups: userGroups,
		},
	}
	sarResult, err := Clientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), &nsListRuleReview, metav1.CreateOptions{})
	if err != nil {
		klog.Errorln(err)
		panic(err)
	}
	if sarResult.Status.Allowed {
		klog.Infoln(" User [ " + userId + " ] has Namespace List Role, Can Access All Namespace")
		nsList, err = Clientset.CoreV1().Namespaces().List(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: labelSelector,
			},
		)
		if err != nil {
			klog.Errorln(err)
			panic(err)
		}
	} else {
		klog.Infoln(" User [ " + userId + " ] has No Namespace List Role, Check If user has Namespace Get Role to Certain Namespace")
		potentialNsList, err := Clientset.CoreV1().Namespaces().List(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: labelSelector,
			},
		)
		if err != nil {
			klog.Errorln(err)
			panic(err)
		}
		var wg sync.WaitGroup
		wg.Add(len(potentialNsList.Items))
		for _, potentialNs := range potentialNsList.Items {
			go func(potentialNs coreApi.Namespace, userId string, userGroups []string, nsList *coreApi.NamespaceList) {
				defer wg.Done()
				nsGetRuleReview := authApi.SubjectAccessReview{
					Spec: authApi.SubjectAccessReviewSpec{
						ResourceAttributes: &authApi.ResourceAttributes{
							Resource:  "namespaces",
							Verb:      "get", //FIXME : list??
							Group:     "",
							Namespace: potentialNs.GetName(),
						},
						User:   userId,
						Groups: userGroups,
					},
				}
				sarResult, err := Clientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), &nsGetRuleReview, metav1.CreateOptions{})
				if err != nil {
					klog.Errorln(err)
					panic(err)
				}
				if sarResult.Status.Allowed {
					klog.Infoln(" User [ " + userId + " ] has Namespace Get Role in Namspace [ " + potentialNs.GetName() + " ]")
					nsList.Items = append(nsList.Items, potentialNs)
				}
			}(potentialNs, userId, userGroups, nsList)
		}
		wg.Wait()

		// if len(nsList.Items) > 0 {
		nsList.APIVersion = potentialNsList.APIVersion
		nsList.Continue = potentialNsList.Continue
		nsList.ResourceVersion = potentialNsList.ResourceVersion
		nsList.TypeMeta = potentialNsList.TypeMeta
		// } else {
		// 	klog.Infoln(" User [ " + userId + " ] has No Namespace Get Role in Any Namspace")
		// }
	}
	// if len(nsList.Items) > 0 {
	// 	klog.Infoln("=== [ " + userId + " ] Accessible Namespace ===")
	// 	for _, ns := range nsList.Items {
	// 		klog.Infoln("  " + ns.Name)
	// 	}
	// }
	return *nsList
}

func Contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	_, ok := set[item]
	return ok
}
