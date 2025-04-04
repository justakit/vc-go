/*
Copyright Gen Digital Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifiable

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/trustbloc/vc-go/dataintegrity"
	"github.com/trustbloc/vc-go/dataintegrity/models"
)

// DataIntegrityProofContext holds parameters for creating or validating a Data Integrity Proof.
type DataIntegrityProofContext struct {
	SigningKeyID string     // eg did:foo:bar#key-1
	ProofPurpose string     // assertionMethod
	CryptoSuite  string     // ecdsa-2019
	Created      *time.Time //
	Expires      *time.Time //
	Domain       string     //
	Challenge    string     //
}

// AddDataIntegrityProof adds a Data Integrity Proof to the Credential.
func (vc *Credential) AddDataIntegrityProof(context *DataIntegrityProofContext, signer *dataintegrity.Signer) error {
	vcBytes, err := vc.MarshalJSON()
	if err != nil {
		return fmt.Errorf("add data integrity proof to VC: %w", err)
	}

	// TODO: rewrite to use json object instead bytes presentation
	proofs, err := addDataIntegrityProof(context, vcBytes, signer)
	if err != nil {
		return err
	}

	vc.ldProofs = proofs

	if len(vc.ldProofs) > 0 {
		vc.credentialJSON[jsonFldLDProof] = proofsToRaw(vc.ldProofs)
	}

	return nil
}

// AddDataIntegrityProof adds a Data Integrity Proof to the Presentation.
func (vp *Presentation) AddDataIntegrityProof(context *DataIntegrityProofContext, signer *dataintegrity.Signer) error {
	vpBytes, err := vp.MarshalJSON()
	if err != nil {
		return fmt.Errorf("add data integrity proof to VP: %w", err)
	}

	proofs, err := addDataIntegrityProof(context, vpBytes, signer)
	if err != nil {
		return err
	}

	vp.Proofs = proofs

	return nil
}

const (
	assertionMethod = "assertionMethod"
)

func addDataIntegrityProof(
	context *DataIntegrityProofContext,
	ldBytes []byte,
	signer *dataintegrity.Signer,
) ([]Proof, error) {
	var createdTime, expiresTime time.Time
	if context.Created == nil {
		createdTime = time.Now()
	} else {
		createdTime = *context.Created
	}

	if context.Expires != nil {
		expiresTime = *context.Expires
	}

	if context.ProofPurpose == "" {
		context.ProofPurpose = assertionMethod
	}

	signed, err := signer.AddProof(ldBytes, &models.ProofOptions{
		Purpose:              context.ProofPurpose,
		VerificationMethodID: context.SigningKeyID,
		ProofType:            models.DataIntegrityProof,
		SuiteType:            context.CryptoSuite,
		Domain:               context.Domain,
		Challenge:            context.Challenge,
		Created:              createdTime,
		Expires:              expiresTime,
	})
	if err != nil {
		return nil, err
	}

	// Get a proof from json-ld document.
	var rProof rawProof

	err = json.Unmarshal(signed, &rProof)
	if err != nil {
		return nil, err
	}

	proofs, err := parseLDProof(rProof.Proof)
	if err != nil {
		return nil, err
	}

	return proofs, nil
}

type verifyDataIntegrityOpts struct {
	Verifier  *dataintegrity.Verifier
	Purpose   string
	Domain    string
	Challenge string
}

// TODO: refactor to directly use map[string]inteface{} instead []byte.
func checkDataIntegrityProof(ldBytes []byte, opts *verifyDataIntegrityOpts) error {
	if opts == nil || opts.Verifier == nil {
		return errors.New("data integrity proof needs data integrity verifier")
	}

	if opts.Purpose == "" {
		opts.Purpose = assertionMethod
	}

	return opts.Verifier.VerifyProof(ldBytes, &models.ProofOptions{
		Purpose:   opts.Purpose,
		ProofType: models.DataIntegrityProof,
		Domain:    opts.Domain,
		Challenge: opts.Challenge,
	})
}
