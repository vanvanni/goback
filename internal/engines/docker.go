package engines

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type Docker struct {
	cli *client.Client
}

type VolumeInfo struct {
	Name       string
	MountPoint string
	Driver     string
	Containers []ContainerVolumeUsage
}

type ContainerVolumeUsage struct {
	ContainerID   string
	ContainerName string
	MountPath     string
}

func NewDocker() (*Docker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Docker{cli: cli}, nil
}

func (d *Docker) Kind() EngineKind {
	return EngineKindDocker
}

func (d *Docker) Close() error {
	return d.cli.Close()
}

func (d *Docker) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	volumeList, err := d.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	containers, err := d.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	volumeInfos := make([]VolumeInfo, 0, len(volumeList.Volumes))

	for _, vol := range volumeList.Volumes {
		info := VolumeInfo{
			Name:       vol.Name,
			MountPoint: vol.Mountpoint,
			Driver:     vol.Driver,
			Containers: []ContainerVolumeUsage{},
		}

		for _, cnt := range containers {
			for _, mount := range cnt.Mounts {
				if mount.Name == vol.Name || mount.Source == vol.Mountpoint {
					containerName := ""
					if len(cnt.Names) > 0 {
						containerName = cnt.Names[0]
						if len(containerName) > 0 && containerName[0] == '/' {
							containerName = containerName[1:]
						}
					}

					info.Containers = append(info.Containers, ContainerVolumeUsage{
						ContainerID:   cnt.ID[:12], // Short ID
						ContainerName: containerName,
						MountPath:     mount.Destination,
					})
				}
			}
		}

		volumeInfos = append(volumeInfos, info)
	}

	return volumeInfos, nil
}

func (d *Docker) Pause(ctx context.Context, containerID string) error {
	err := d.cli.ContainerPause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to pause container %s: %w", containerID, err)
	}
	return nil
}

func (d *Docker) Unpause(ctx context.Context, containerID string) error {
	err := d.cli.ContainerUnpause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to unpause container %s: %w", containerID, err)
	}
	return nil
}

func (d *Docker) GetVolumeByName(ctx context.Context, name string) (*VolumeInfo, error) {
	volumes, err := d.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}

	for _, vol := range volumes {
		if vol.Name == name {
			return &vol, nil
		}
	}

	return nil, fmt.Errorf("volume %s not found", name)
}

func (d *Docker) GetContainersByVolume(ctx context.Context, volumeName string) ([]ContainerVolumeUsage, error) {
	vol, err := d.GetVolumeByName(ctx, volumeName)
	if err != nil {
		return nil, err
	}
	return vol.Containers, nil
}

func (d *Docker) WaitForPaused(ctx context.Context, containerID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			inspect, err := d.cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			if inspect.State.Paused {
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

func (d *Docker) WaitForAllPaused(ctx context.Context, containers []ContainerVolumeUsage) error {
	for _, container := range containers {
		if err := d.WaitForPaused(ctx, container.ContainerID); err != nil {
			return fmt.Errorf("failed waiting for container %s (%s): %w",
				container.ContainerName, container.ContainerID, err)
		}
	}
	return nil
}
