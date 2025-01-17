/*
 * HCS API
 *
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * API version: 2.1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package hcsschema

// Options for HcsPauseComputeSystem
type PauseOptions struct {
	SuspensionLevel string `json:"SuspensionLevel,omitempty"`

	HostedNotification *PauseNotification `json:"HostedNotification,omitempty"`
}
