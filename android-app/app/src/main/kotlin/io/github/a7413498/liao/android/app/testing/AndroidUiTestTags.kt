package io.github.a7413498.liao.android.app.testing

object LoginTestTags {
    const val TITLE = "login_title"
    const val DESCRIPTION = "login_description"
    const val BASE_URL_INPUT = "login_base_url_input"
    const val ACCESS_CODE_INPUT = "login_access_code_input"
    const val LOGIN_BUTTON = "login_button"
    const val LOADING_INDICATOR = "login_loading_indicator"
}

object IdentityTestTags {
    const val TITLE = "identity_title"
    const val DESCRIPTION = "identity_description"
    const val NAME_INPUT = "identity_name_input"
    const val SEX_INPUT = "identity_sex_input"
    const val PRIMARY_ACTION_BUTTON = "identity_primary_action_button"
    const val SECONDARY_ACTION_BUTTON = "identity_secondary_action_button"
    const val LOADING_INDICATOR = "identity_loading_indicator"
    const val EMPTY_STATE = "identity_empty_state"
    const val LIST = "identity_list"
    const val DELETE_DIALOG_CONFIRM = "identity_delete_dialog_confirm"
    const val DELETE_DIALOG_CANCEL = "identity_delete_dialog_cancel"

    fun itemCard(id: String): String = "identity_item_card_$id"
    fun editButton(id: String): String = "identity_item_edit_$id"
    fun deleteButton(id: String): String = "identity_item_delete_$id"
    fun selectButton(id: String): String = "identity_item_select_$id"
}

object ChatListTestTags {
    const val HISTORY_TAB = "chatlist_history_tab"
    const val FAVORITE_TAB = "chatlist_favorite_tab"
    const val SETTINGS_BUTTON = "chatlist_settings_button"
    const val TOP_GLOBAL_FAVORITES_BUTTON = "chatlist_top_global_favorites_button"
    const val QUICK_GLOBAL_FAVORITES_BUTTON = "chatlist_quick_global_favorites_button"
    const val REFRESH_BUTTON = "chatlist_refresh_button"
    const val LOADING_INDICATOR = "chatlist_loading_indicator"
    const val STATE_CARD = "chatlist_state_card"
    const val LIST = "chatlist_list"

    fun item(id: String): String = "chatlist_item_$id"
}

object SettingsTestTags {
    const val BACK_BUTTON = "settings_back_button"
    const val THEME_SUMMARY = "settings_theme_summary"
    const val THEME_AUTO_BUTTON = "settings_theme_auto_button"
    const val THEME_LIGHT_BUTTON = "settings_theme_light_button"
    const val THEME_DARK_BUTTON = "settings_theme_dark_button"
    const val BASE_URL_INPUT = "settings_base_url_input"
    const val SAVE_BASE_URL_BUTTON = "settings_save_base_url_button"
    const val OPEN_GLOBAL_FAVORITES_BUTTON = "settings_open_global_favorites_button"
    const val OPEN_MEDIA_LIBRARY_BUTTON = "settings_open_media_library_button"
    const val OPEN_MTPHOTO_BUTTON = "settings_open_mtphoto_button"
    const val OPEN_DOUYIN_BUTTON = "settings_open_douyin_button"
    const val OPEN_VIDEO_EXTRACT_BUTTON = "settings_open_video_extract_button"
    const val OPEN_VIDEO_EXTRACT_TASKS_BUTTON = "settings_open_video_extract_tasks_button"
    const val LOGOUT_BUTTON = "settings_logout_button"
    const val IMAGE_PORT_MODE_FIXED_BUTTON = "settings_image_port_mode_fixed_button"
    const val IMAGE_PORT_MODE_PROBE_BUTTON = "settings_image_port_mode_probe_button"
    const val IMAGE_PORT_MODE_REAL_BUTTON = "settings_image_port_mode_real_button"
    const val SAVE_SYSTEM_CONFIG_BUTTON = "settings_save_system_config_button"
}
