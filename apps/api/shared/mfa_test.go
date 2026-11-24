package shared

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestMFA_GenerateAndValidateTOTP(t *testing.T) {
	key, err := GenerateTOTPSecret("test@example.com")
	if err != nil {
		t.Fatalf("failed to generate TOTP secret: %v", err)
	}

	if key.Secret() == "" {
		t.Errorf("expected non-empty secret")
	}

	if key.URL() == "" {
		t.Errorf("expected non-empty otpauth URL")
	}

	// Generate valid passcode for current time
	passcode, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("failed to generate passcode: %v", err)
	}

	// Validate valid passcode
	if !ValidateTOTPCode(passcode, key.Secret()) {
		t.Errorf("expected valid passcode to pass verification")
	}

	// Validate invalid passcode
	if ValidateTOTPCode("000000", key.Secret()) && passcode != "000000" {
		t.Errorf("expected bogus passcode to fail verification")
	}

	// Empty inputs
	if ValidateTOTPCode("", key.Secret()) || ValidateTOTPCode(passcode, "") {
		t.Errorf("expected empty inputs to fail")
	}
}

func TestMFA_RecoveryCodes(t *testing.T) {
	plainCodes, hashedCodes, err := GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatalf("failed to generate recovery codes: %v", err)
	}

	if len(plainCodes) != 8 || len(hashedCodes) != 8 {
		t.Fatalf("expected 8 recovery codes, got %d and %d", len(plainCodes), len(hashedCodes))
	}

	// Test first code consumption
	validCode := plainCodes[0]
	consumed, remaining := ValidateAndConsumeRecoveryCode(validCode, hashedCodes)
	if !consumed {
		t.Errorf("expected valid recovery code to be consumed")
	}
	if len(remaining) != 7 {
		t.Errorf("expected 7 remaining codes, got %d", len(remaining))
	}

	// Attempting to consume the same code again should fail
	consumedAgain, _ := ValidateAndConsumeRecoveryCode(validCode, remaining)
	if consumedAgain {
		t.Errorf("expected consumed recovery code to not be reusable")
	}

	// Attempting to consume an invalid code
	consumedInvalid, _ := ValidateAndConsumeRecoveryCode("INVALID-CODE", remaining)
	if consumedInvalid {
		t.Errorf("expected invalid code to fail")
	}
}
