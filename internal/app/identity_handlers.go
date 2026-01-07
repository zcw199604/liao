package app

import (
	"net/http"
	"strings"
)

func (a *App) handleGetIdentityList(w http.ResponseWriter, r *http.Request) {
	identities, err := a.identityService.GetAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code": -1,
			"msg":  "查询失败",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": identities,
	})
}

func (a *App) handleCreateIdentity(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	sex := strings.TrimSpace(r.FormValue("sex"))

	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "名字不能为空"})
		return
	}
	if sex != "男" && sex != "女" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "性别必须是男或女"})
		return
	}

	identity, err := a.identityService.Create(r.Context(), name, sex)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "创建失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": identity,
	})
}

func (a *App) handleQuickCreateIdentity(w http.ResponseWriter, r *http.Request) {
	identity, err := a.identityService.QuickCreate(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "创建失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": identity,
	})
}

func (a *App) handleUpdateIdentity(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id := strings.TrimSpace(r.FormValue("id"))
	name := strings.TrimSpace(r.FormValue("name"))
	sex := strings.TrimSpace(r.FormValue("sex"))

	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "身份ID不能为空"})
		return
	}
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "名字不能为空"})
		return
	}
	if sex != "男" && sex != "女" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "性别必须是男或女"})
		return
	}

	identity, err := a.identityService.Update(r.Context(), id, name, sex)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "更新失败"})
		return
	}
	if identity == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "身份不存在"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": identity,
	})
}

func (a *App) handleUpdateIdentityID(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	oldID := strings.TrimSpace(r.FormValue("oldId"))
	newID := strings.TrimSpace(r.FormValue("newId"))
	name := strings.TrimSpace(r.FormValue("name"))
	sex := strings.TrimSpace(r.FormValue("sex"))

	if oldID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "旧身份ID不能为空"})
		return
	}
	if newID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "新身份ID不能为空"})
		return
	}
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "名字不能为空"})
		return
	}
	if sex != "男" && sex != "女" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "性别必须是男或女"})
		return
	}

	identity, err := a.identityService.UpdateID(r.Context(), oldID, newID, name, sex)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "更新失败"})
		return
	}
	if identity == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "更新失败，可能旧身份不存在或新ID已被使用"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": identity,
	})
}

func (a *App) handleDeleteIdentity(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "身份ID不能为空"})
		return
	}

	ok, err := a.identityService.Delete(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "删除失败"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "身份不存在"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "msg": "success"})
}

func (a *App) handleSelectIdentity(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "身份ID不能为空"})
		return
	}

	identity, err := a.identityService.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "查询失败"})
		return
	}
	if identity == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "身份不存在"})
		return
	}

	_ = a.identityService.UpdateLastUsedAt(r.Context(), id)

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": identity,
	})
}

