package pwd

import "golang.org/x/crypto/bcrypt"

// HashPwd把明文密码加密成哈希
func HashPwd(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPwd校验明文密码是否和哈希匹配
func CheckPwd(hash string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}