// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlflow

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	commonplugins "github.com/kubeflow/pipelines/backend/src/common/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateSelfSignedCertPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-ca"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

func TestLoadCertsFromDir_NonexistentDir(t *testing.T) {
	result := loadCertsFromDir("/nonexistent/path/that/does/not/exist")
	assert.Nil(t, result)
}

func TestLoadCertsFromDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := loadCertsFromDir(dir)
	assert.Nil(t, result)
}

func TestLoadCertsFromDir_SkipsNonCertFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a cert"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("key: value"), 0644))
	result := loadCertsFromDir(dir)
	assert.Nil(t, result)
}

func TestLoadCertsFromDir_LoadsCrtAndPemFiles(t *testing.T) {
	dir := t.TempDir()
	certPEM := generateSelfSignedCertPEM(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "service-ca.crt"), certPEM, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.pem"), certPEM, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("not a cert"), 0644))

	result := loadCertsFromDir(dir)
	require.NotNil(t, result)
	assert.Greater(t, len(result), 0)

	pool := x509.NewCertPool()
	assert.True(t, pool.AppendCertsFromPEM(result))
}

func TestLoadCertsFromDir_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir.crt")
	require.NoError(t, os.Mkdir(subdir, 0755))

	result := loadCertsFromDir(dir)
	assert.Nil(t, result)
}

func TestBuildHTTPClient_NilTLSConfig_ProbesDefaultDir(t *testing.T) {
	dir := t.TempDir()
	certPEM := generateSelfSignedCertPEM(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ca.crt"), certPEM, 0644))

	client, err := buildHTTPClientWithDefaultCADir(5*time.Second, nil, dir)
	require.NoError(t, err)
	require.NotNil(t, client)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "transport should be *http.Transport")
	require.NotNil(t, transport.TLSClientConfig, "TLSClientConfig should be set")
	assert.NotNil(t, transport.TLSClientConfig.RootCAs, "RootCAs should contain the discovered certificate")
}

func TestBuildHTTPClient_ExplicitCABundlePath(t *testing.T) {
	dir := t.TempDir()
	certPEM := generateSelfSignedCertPEM(t)
	caPath := filepath.Join(dir, "ca-bundle.crt")
	require.NoError(t, os.WriteFile(caPath, certPEM, 0644))

	client, err := BuildHTTPClient(5*time.Second, &commonplugins.TLSConfig{
		CABundlePath: caPath,
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestBuildHTTPClient_ExplicitCABundlePath_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "bad.crt")
	require.NoError(t, os.WriteFile(caPath, []byte("not a cert"), 0644))

	_, err := BuildHTTPClient(5*time.Second, &commonplugins.TLSConfig{
		CABundlePath: caPath,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not contain valid PEM certificates")
}

func TestBuildHTTPClient_ExplicitCABundlePath_FileNotFound(t *testing.T) {
	_, err := BuildHTTPClient(5*time.Second, &commonplugins.TLSConfig{
		CABundlePath: "/nonexistent/ca.crt",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestBuildHTTPClient_NilTLSConfig_Succeeds(t *testing.T) {
	client, err := BuildHTTPClient(5*time.Second, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestBuildHTTPClient_EmptyTLSConfig_Succeeds(t *testing.T) {
	client, err := BuildHTTPClient(5*time.Second, &commonplugins.TLSConfig{})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestBuildHTTPClient_InsecureSkipVerify_Rejected(t *testing.T) {
	_, err := BuildHTTPClient(5*time.Second, &commonplugins.TLSConfig{
		InsecureSkipVerify: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insecureSkipVerify is not supported")
}

func TestBuildHTTPClient_DefaultDir_MalformedCerts_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.crt"), []byte("not valid PEM data"), 0644))

	_, err := buildHTTPClientWithDefaultCADir(5*time.Second, nil, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not contain valid PEM certificates")
}

func TestBuildHTTPClient_DefaultDir_EmptyDir_Succeeds(t *testing.T) {
	dir := t.TempDir()

	client, err := buildHTTPClientWithDefaultCADir(5*time.Second, nil, dir)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestBuildHTTPClient_DefaultDir_NonexistentDir_Succeeds(t *testing.T) {
	client, err := buildHTTPClientWithDefaultCADir(5*time.Second, nil, "/nonexistent/ca/dir")
	require.NoError(t, err)
	assert.NotNil(t, client)
}
