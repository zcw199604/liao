package app

import (
	"net/http"
	"strings"
)

func (a *App) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code": -1,
			"msg":  "访问码不能为空",
		})
		return
	}

	accessCode := r.FormValue("accessCode")
	if strings.TrimSpace(accessCode) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code": -1,
			"msg":  "访问码不能为空",
		})
		return
	}

	if accessCode != a.cfg.AuthAccessCode {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code": -1,
			"msg":  "访问码错误",
		})
		return
	}

	token, err := a.jwt.GenerateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code": -1,
			"msg":  "登录失败",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code":  0,
		"msg":   "登录成功",
		"token": token,
	})
}

func (a *App) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSON(w, http.StatusOK, map[string]any{
			"code": -1,
			"msg":  "Token缺失",
		})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	valid := a.jwt.ValidateToken(tokenString)

	writeJSON(w, http.StatusOK, map[string]any{
		"code":  ifThenElse(valid, 0, -1),
		"msg":   ifThenElse(valid, "Token有效", "Token无效"),
		"valid": valid,
	})
}

func ifThenElse[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
