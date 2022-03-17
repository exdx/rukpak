package decorator

import (
	"fmt"
	"strings"

	"github.com/operator-framework/rukpak/api/v1alpha1"
	"github.com/operator-framework/rukpak/internal/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BundleContentDecorator knows how to build a bundle unpacking pod given a Bundle.
type BundleContentDecorator interface {
	BuildPod(bundle *v1alpha1.Bundle) *corev1.Pod
}

type ImageDecorator struct {
	bundle      *v1alpha1.Bundle
	ns          string
	unpackImage string
}

type GitDecorator struct {
	bundle      *v1alpha1.Bundle
	ns          string
	unpackImage string
}

func NewImageDecorator(bundle *v1alpha1.Bundle, namespace, unpackImage string) BundleContentDecorator {
	return &ImageDecorator{
		bundle:      bundle,
		ns:          namespace,
		unpackImage: unpackImage,
	}
}

func NewGitDecorator(bundle *v1alpha1.Bundle, namespace, unpackImage string) BundleContentDecorator {
	return &GitDecorator{
		bundle:      bundle,
		ns:          namespace,
		unpackImage: unpackImage,
	}
}

func (i *ImageDecorator) BuildPod(bundle *v1alpha1.Bundle) *corev1.Pod {
	pod := &corev1.Pod{}
	pod = common(bundle, pod, i.ns, i.unpackImage)
	pod.Spec.Containers[0].Image = bundle.Spec.Source.Image.Ref

	return pod
}

func (g *GitDecorator) BuildPod(bundle *v1alpha1.Bundle) *corev1.Pod {
	pod := &corev1.Pod{}
	pod = common(bundle, pod, g.ns, g.unpackImage)

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "manifests", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	})
	// Note: use host networking (hack) so we can resolve https://github.com references in the
	// Bundle's spec.Source.Git.Repository. Maybe we should revisit having the user inline https://github.com?
	pod.Spec.HostNetwork = true

	// TODO: length check
	// TODO: function responsible for determine which ref to use
	// TODO: bundle unpack failing silently when pod log stream is empty (e.g. `{}`)
	// TODO: bundle has an image custom column -- maybe we need a source type configuration instead?
	// TODO: bundleinstance reporting a successful installation state despite installing nothing.
	source := bundle.Spec.Source.Git
	repository := source.Repository
	directory := "./manifests"
	if source.Directory != "" {
		directory = source.Directory
	}
	checkedCommand := fmt.Sprintf("git clone %s && cd %s && git checkout %s && cp -r %s/* /manifests", repository, strings.Split(repository, "/")[4], source.Ref.Commit, directory)
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
		Name:            "clone-repository",
		Image:           "bitnami/git:latest",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/bash", "-c", checkedCommand},
		VolumeMounts:    []corev1.VolumeMount{{Name: "util", MountPath: "/util"}, {Name: "manifests", MountPath: "/manifests"}},
	})

	pod.Spec.Containers[0].Image = "bitnami/git:latest"

	return pod
}

// common includes pod configuration logic that both the ImageDecorator and the GitDecorator pods support.
func common(bundle *v1alpha1.Bundle, pod *corev1.Pod, namespace string, unpackImage string) *corev1.Pod {
	controllerRef := metav1.NewControllerRef(bundle, bundle.GroupVersionKind())
	automountServiceAccountToken := false
	pod.SetName(util.PodName(util.PlainBundleProvisionerName, bundle.Name))
	pod.SetNamespace(namespace)

	pod.SetLabels(map[string]string{
		"core.rukpak.io/owner-kind": bundle.Kind,
		"core.rukpak.io/owner-name": bundle.Name,
	})

	pod.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
	pod.Spec.AutomountServiceAccountToken = &automountServiceAccountToken

	pod.Spec.Volumes = []corev1.Volume{
		{Name: "util", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}

	pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	pod.Spec.InitContainers = make([]corev1.Container, 1)
	pod.Spec.InitContainers[0].Name = "install-unpack"
	pod.Spec.InitContainers[0].ImagePullPolicy = corev1.PullIfNotPresent
	pod.Spec.InitContainers[0].Image = unpackImage
	pod.Spec.InitContainers[0].Command = []string{"cp", "-Rv", "/unpack", "/util/unpack"}
	pod.Spec.InitContainers[0].VolumeMounts = []corev1.VolumeMount{{Name: "util", MountPath: "/util"}}

	pod.Spec.Containers = make([]corev1.Container, 1)
	pod.Spec.Containers[0].Name = util.BundleUnpackContainerName
	pod.Spec.Containers[0].ImagePullPolicy = corev1.PullAlways
	pod.Spec.Containers[0].Command = []string{"/util/unpack", "--bundle-dir", "/manifests"}
	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "util", MountPath: "/util"}}

	return pod
}
