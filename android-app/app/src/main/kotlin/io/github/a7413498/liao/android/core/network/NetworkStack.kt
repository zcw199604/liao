/*
 * 网络栈集中定义 Retrofit Service、动态 Base URL、JWT 注入和 Hilt 提供器。
 * 其目标是复用现有 Go 服务端接口风格，而不是强行统一成单一响应协议。
 */
package io.github.a7413498.liao.android.core.network

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import okhttp3.Authenticator
import okhttp3.FormBody
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.ResponseBody
import okhttp3.Route
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.Body
import retrofit2.http.Field
import retrofit2.http.FormUrlEncoded
import retrofit2.http.GET
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part
import retrofit2.http.Query
import retrofit2.http.Streaming
import retrofit2.http.Url
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody

@Singleton
class BaseUrlProvider @Inject constructor(
    private val preferencesStore: AppPreferencesStore,
) {
    fun currentApiBaseUrl(): HttpUrl {
        val raw = runBlocking { preferencesStore.readBaseUrl() }
        val normalized = when {
            raw.endsWith("/api/") -> raw
            raw.endsWith("/api") -> "$raw/"
            raw.endsWith("/") -> "${raw}api/"
            else -> "$raw/api/"
        }
        return normalized.toHttpUrl()
    }

    fun currentWebSocketUrl(token: String): String {
        val api = currentApiBaseUrl()
        val scheme = if (api.isHttps) "wss" else "ws"
        val wsPath = api.encodedPath.removeSuffix("/").removeSuffix("/api") + "/ws"
        return api.newBuilder()
            .scheme(scheme)
            .encodedPath(wsPath.ifBlank { "/ws" })
            .setQueryParameter("token", token)
            .build()
            .toString()
    }
}

@Singleton
class DynamicBaseUrlInterceptor @Inject constructor(
    private val baseUrlProvider: BaseUrlProvider,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val original = chain.request()
        val placeholderPrefix = "/api"
        val dynamicBase = baseUrlProvider.currentApiBaseUrl()
        val originalPath = original.url.encodedPath
        val relativePath = originalPath.removePrefix(placeholderPrefix).trimStart('/')
        val newUrlBuilder = dynamicBase.newBuilder()
        if (relativePath.isNotBlank()) {
            newUrlBuilder.addEncodedPathSegments(relativePath)
        }
        val newUrl = newUrlBuilder.encodedQuery(original.url.encodedQuery).build()
        val newRequest = original.newBuilder().url(newUrl).build()
        return chain.proceed(newRequest)
    }
}

@Singleton
class AuthInterceptor @Inject constructor(
    private val preferencesStore: AppPreferencesStore,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val token = runBlocking { preferencesStore.readAuthToken() }
        val request = if (token.isNullOrBlank()) {
            chain.request()
        } else {
            chain.request().newBuilder()
                .header("Authorization", "Bearer $token")
                .build()
        }
        return chain.proceed(request)
    }
}

@Singleton
class TokenAuthenticator @Inject constructor(
    private val preferencesStore: AppPreferencesStore,
) : Authenticator {
    override fun authenticate(route: Route?, response: Response): Request? {
        if (response.request.header("Authorization").isNullOrBlank()) return null
        runBlocking { preferencesStore.clearAuthToken() }
        return null
    }
}

interface AuthApiService {
    @FormUrlEncoded
    @POST("/api/auth/login")
    suspend fun login(@Field("accessCode") accessCode: String): ApiEnvelope<Unit>

    @GET("/api/auth/verify")
    suspend fun verify(): ApiEnvelope<Unit>
}

interface IdentityApiService {
    @GET("/api/getIdentityList")
    suspend fun getIdentityList(): ApiEnvelope<List<IdentityDto>>

    @FormUrlEncoded
    @POST("/api/createIdentity")
    suspend fun createIdentity(
        @Field("name") name: String,
        @Field("sex") sex: String,
    ): ApiEnvelope<IdentityDto>

    @POST("/api/quickCreateIdentity")
    suspend fun quickCreateIdentity(): ApiEnvelope<IdentityDto>

    @FormUrlEncoded
    @POST("/api/deleteIdentity")
    suspend fun deleteIdentity(@Field("id") id: String): ApiEnvelope<Unit>

    @POST("/api/selectIdentity")
    suspend fun selectIdentity(@Query("id") id: String): ApiEnvelope<IdentityDto>

    @FormUrlEncoded
    @POST("/api/updateIdentity")
    suspend fun updateIdentity(
        @Field("id") id: String,
        @Field("name") name: String,
        @Field("sex") sex: String,
    ): ApiEnvelope<IdentityDto>

    @FormUrlEncoded
    @POST("/api/updateIdentityId")
    suspend fun updateIdentityId(
        @Field("oldId") oldId: String,
        @Field("newId") newId: String,
        @Field("name") name: String,
        @Field("sex") sex: String,
    ): ApiEnvelope<IdentityDto>
}

interface ChatApiService {
    @FormUrlEncoded
    @POST("/api/getHistoryUserList")
    suspend fun getHistoryUserList(
        @Field("myUserID") myUserId: String,
        @Field("vipcode") vipCode: String = "",
        @Field("serverPort") serverPort: String = "1001",
        @Field("cookieData") cookieData: String,
        @Field("referer") referer: String,
        @Field("userAgent") userAgent: String,
    ): List<ChatUserDto>

    @FormUrlEncoded
    @POST("/api/getFavoriteUserList")
    suspend fun getFavoriteUserList(
        @Field("myUserID") myUserId: String,
        @Field("vipcode") vipCode: String = "",
        @Field("serverPort") serverPort: String = "1001",
        @Field("cookieData") cookieData: String,
        @Field("referer") referer: String,
        @Field("userAgent") userAgent: String,
    ): List<ChatUserDto>

    @FormUrlEncoded
    @POST("/api/getMessageHistory")
    suspend fun getMessageHistory(
        @Field("myUserID") myUserId: String,
        @Field("UserToID") userToId: String,
        @Field("isFirst") isFirst: String,
        @Field("firstTid") firstTid: String,
        @Field("vipcode") vipCode: String = "",
        @Field("serverPort") serverPort: String = "1001",
        @Field("cookieData") cookieData: String,
        @Field("referer") referer: String,
        @Field("userAgent") userAgent: String,
    ): List<ChatMessageDto>

    @FormUrlEncoded
    @POST("/api/reportReferrer")
    suspend fun reportReferrer(
        @Field("referrerUrl") referrerUrl: String,
        @Field("currUrl") currUrl: String,
        @Field("userid") userId: String,
        @Field("cookieData") cookieData: String,
        @Field("referer") referer: String,
        @Field("userAgent") userAgent: String,
    ): ApiEnvelope<Unit>

    @FormUrlEncoded
    @POST("/api/toggleFavorite")
    suspend fun toggleFavorite(
        @Field("myUserID") myUserId: String,
        @Field("UserToID") userToId: String,
        @Field("vipcode") vipCode: String = "",
        @Field("serverPort") serverPort: String = "1001",
        @Field("cookieData") cookieData: String,
        @Field("referer") referer: String,
        @Field("userAgent") userAgent: String,
    ): ApiEnvelope<Unit>

    @FormUrlEncoded
    @POST("/api/cancelFavorite")
    suspend fun cancelFavorite(
        @Field("myUserID") myUserId: String,
        @Field("UserToID") userToId: String,
        @Field("vipcode") vipCode: String = "",
        @Field("serverPort") serverPort: String = "1001",
        @Field("cookieData") cookieData: String,
        @Field("referer") referer: String,
        @Field("userAgent") userAgent: String,
    ): ApiEnvelope<Unit>
}

interface FavoriteApiService {
    @FormUrlEncoded
    @POST("/api/favorite/add")
    suspend fun addFavorite(
        @Field("identityId") identityId: String,
        @Field("targetUserId") targetUserId: String,
        @Field("targetUserName") targetUserName: String,
    ): ApiEnvelope<JsonElement>

    @GET("/api/favorite/listAll")
    suspend fun listAllFavorites(): ApiEnvelope<List<JsonElement>>
}

interface MediaApiService {
    @Multipart
    @POST("/api/uploadMedia")
    suspend fun uploadMedia(@Part file: MultipartBody.Part): ResponseBody

    @GET("/api/getAllUploadImages")
    suspend fun getAllUploadImages(
        @Query("page") page: Int,
        @Query("pageSize") pageSize: Int,
        @Query("source") source: String? = null,
        @Query("douyinSecUserId") douyinSecUserId: String? = null,
    ): JsonElement

    @GET("/api/getChatImages")
    suspend fun getChatImages(
        @Query("userId1") userId1: String,
        @Query("userId2") userId2: String,
        @Query("limit") limit: Int = 20,
    ): JsonElement

    @FormUrlEncoded
    @POST("/api/recordImageSend")
    suspend fun recordImageSend(
        @Field("remoteUrl") remoteUrl: String,
        @Field("fromUserId") fromUserId: String,
        @Field("toUserId") toUserId: String,
        @Field("localFilename") localFilename: String = "",
    ): ApiEnvelope<Unit>

    @POST("/api/batchDeleteMedia")
    suspend fun batchDeleteMedia(@Body payload: JsonElement): ApiEnvelope<Unit>
}

interface MtPhotoApiService {
    @GET("/api/getMtPhotoAlbums")
    suspend fun getAlbums(): JsonElement

    @GET("/api/getMtPhotoAlbumFiles")
    suspend fun getAlbumFiles(@Query("albumId") albumId: Long): JsonElement

    @GET("/api/getMtPhotoFolderRoot")
    suspend fun getFolderRoot(): JsonElement

    @GET("/api/getMtPhotoFolderContent")
    suspend fun getFolderContent(@Query("folderId") folderId: Long): JsonElement
}

interface DouyinApiService {
    @POST("/api/douyin/detail")
    suspend fun getDetail(@Body payload: JsonElement): JsonElement

    @POST("/api/douyin/account")
    suspend fun getAccount(@Body payload: JsonElement): JsonElement

    @Streaming
    @GET
    suspend fun downloadFile(@Url url: String): ResponseBody
}

interface SystemApiService {
    @GET("/api/getConnectionStats")
    suspend fun getConnectionStats(): ApiEnvelope<ConnectionStatsDto>

    @GET("/api/getSystemConfig")
    suspend fun getSystemConfig(): ApiEnvelope<SystemConfigDto>

    @POST("/api/updateSystemConfig")
    suspend fun updateSystemConfig(@Body payload: JsonElement): ApiEnvelope<SystemConfigDto>

    @POST("/api/resolveImagePort")
    suspend fun resolveImagePort(@Body payload: JsonElement): ApiEnvelope<JsonElement>
}

interface VideoExtractApiService {
    @Multipart
    @POST("/api/uploadVideoExtractInput")
    suspend fun uploadVideoExtractInput(@Part file: MultipartBody.Part): ApiEnvelope<JsonElement>

    @GET("/api/probeVideo")
    suspend fun probeVideo(
        @Query("sourceType") sourceType: String,
        @Query("localPath") localPath: String? = null,
        @Query("md5") md5: String? = null,
    ): ApiEnvelope<JsonElement>

    @POST("/api/createVideoExtractTask")
    suspend fun createTask(@Body payload: JsonElement): ApiEnvelope<JsonElement>
}

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {
    @Provides
    @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        isLenient = true
    }

    @Provides
    @Singleton
    fun provideLoggingInterceptor(): HttpLoggingInterceptor =
        HttpLoggingInterceptor().apply { level = HttpLoggingInterceptor.Level.BASIC }

    @Provides
    @Singleton
    fun provideOkHttpClient(
        loggingInterceptor: HttpLoggingInterceptor,
        dynamicBaseUrlInterceptor: DynamicBaseUrlInterceptor,
        authInterceptor: AuthInterceptor,
        tokenAuthenticator: TokenAuthenticator,
    ): OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(20, TimeUnit.SECONDS)
        .readTimeout(65, TimeUnit.SECONDS)
        .writeTimeout(65, TimeUnit.SECONDS)
        .addInterceptor(dynamicBaseUrlInterceptor)
        .addInterceptor(authInterceptor)
        .addInterceptor(loggingInterceptor)
        .authenticator(tokenAuthenticator)
        .build()

    @Provides
    @Singleton
    fun provideRetrofit(
        okHttpClient: OkHttpClient,
        json: Json,
    ): Retrofit = Retrofit.Builder()
        .baseUrl("https://placeholder.invalid/api/")
        .client(okHttpClient)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()

    @Provides
    @Singleton
    fun provideAuthApiService(retrofit: Retrofit): AuthApiService = retrofit.create(AuthApiService::class.java)

    @Provides
    @Singleton
    fun provideIdentityApiService(retrofit: Retrofit): IdentityApiService = retrofit.create(IdentityApiService::class.java)

    @Provides
    @Singleton
    fun provideChatApiService(retrofit: Retrofit): ChatApiService = retrofit.create(ChatApiService::class.java)

    @Provides
    @Singleton
    fun provideFavoriteApiService(retrofit: Retrofit): FavoriteApiService = retrofit.create(FavoriteApiService::class.java)

    @Provides
    @Singleton
    fun provideMediaApiService(retrofit: Retrofit): MediaApiService = retrofit.create(MediaApiService::class.java)

    @Provides
    @Singleton
    fun provideMtPhotoApiService(retrofit: Retrofit): MtPhotoApiService = retrofit.create(MtPhotoApiService::class.java)

    @Provides
    @Singleton
    fun provideDouyinApiService(retrofit: Retrofit): DouyinApiService = retrofit.create(DouyinApiService::class.java)

    @Provides
    @Singleton
    fun provideSystemApiService(retrofit: Retrofit): SystemApiService = retrofit.create(SystemApiService::class.java)

    @Provides
    @Singleton
    fun provideVideoExtractApiService(retrofit: Retrofit): VideoExtractApiService = retrofit.create(VideoExtractApiService::class.java)
}
