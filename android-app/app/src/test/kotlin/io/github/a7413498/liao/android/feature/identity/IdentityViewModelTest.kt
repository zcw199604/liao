package io.github.a7413498.liao.android.feature.identity

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class IdentityViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<IdentityRepository>()

    @Test
    fun `init should load identities into ui state`() = runTest(mainDispatcherRule.dispatcher) {
        val identity = sampleIdentity(id = "identity-1", name = "Alice")
        coEvery { repository.loadIdentities() } returns AppResult.Success(listOf(identity))

        val viewModel = IdentityViewModel(repository)
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.loading)
        assertEquals(listOf(identity), viewModel.uiState.identities)
        assertNull(viewModel.uiState.message)
    }

    @Test
    fun `quick create should stop when editing is active`() = runTest(mainDispatcherRule.dispatcher) {
        val editing = sampleIdentity(id = "identity-1", name = "Alice")
        coEvery { repository.loadIdentities() } returns AppResult.Success(listOf(editing))

        val viewModel = IdentityViewModel(repository)
        advanceUntilIdle()
        viewModel.startEditing(editing)

        viewModel.quickCreate()
        advanceUntilIdle()

        assertEquals("请先完成或取消当前编辑", viewModel.uiState.message)
        coVerify(exactly = 0) { repository.quickCreate() }
    }

    @Test
    fun `submit identity should validate blank name and sex`() = runTest(mainDispatcherRule.dispatcher) {
        coEvery { repository.loadIdentities() } returns AppResult.Success(emptyList())
        val viewModel = IdentityViewModel(repository)
        advanceUntilIdle()

        viewModel.submitIdentity()
        assertEquals("名字不能为空", viewModel.uiState.message)

        viewModel.updateName("Alice")
        viewModel.updateSex("   ")
        viewModel.submitIdentity()
        assertEquals("性别不能为空", viewModel.uiState.message)
    }

    @Test
    fun `submit identity should create new identity and clear form on success`() = runTest(mainDispatcherRule.dispatcher) {
        val created = sampleIdentity(id = "identity-2", name = "Bob")
        coEvery { repository.loadIdentities() } returnsMany listOf(
            AppResult.Success(emptyList()),
            AppResult.Success(listOf(created)),
        )
        coEvery { repository.createIdentity("Bob", "男") } returns AppResult.Success(created)

        val viewModel = IdentityViewModel(repository)
        advanceUntilIdle()
        viewModel.updateName("Bob")
        viewModel.updateSex("男")

        viewModel.submitIdentity()
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.loading)
        assertEquals(false, viewModel.uiState.saving)
        assertEquals(listOf(created), viewModel.uiState.identities)
        assertEquals("", viewModel.uiState.nameInput)
        assertEquals("女", viewModel.uiState.sexInput)
        assertNull(viewModel.uiState.editingIdentityId)
        assertEquals("已创建 Bob", viewModel.uiState.message)
    }

    @Test
    fun `submit identity should update existing identity and surface repository error`() = runTest(mainDispatcherRule.dispatcher) {
        val original = sampleIdentity(id = "identity-1", name = "Alice")
        coEvery { repository.loadIdentities() } returns AppResult.Success(listOf(original))
        coEvery { repository.updateIdentity("identity-1", "Alice2", "男") } returns AppResult.Error("更新失败")

        val viewModel = IdentityViewModel(repository)
        advanceUntilIdle()
        viewModel.startEditing(original)
        viewModel.updateName("Alice2")
        viewModel.updateSex("男")

        viewModel.submitIdentity()
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.saving)
        assertEquals("更新失败", viewModel.uiState.message)
        assertEquals("identity-1", viewModel.uiState.editingIdentityId)
    }

    @Test
    fun `delete confirmed should clear form when removing active editing identity`() = runTest(mainDispatcherRule.dispatcher) {
        val target = sampleIdentity(id = "identity-1", name = "Alice")
        coEvery { repository.loadIdentities() } returnsMany listOf(
            AppResult.Success(listOf(target)),
            AppResult.Success(emptyList()),
        )
        coEvery { repository.deleteIdentity(target.id) } returns AppResult.Success(Unit)

        val viewModel = IdentityViewModel(repository)
        advanceUntilIdle()
        viewModel.startEditing(target)
        viewModel.confirmDelete(target)

        viewModel.deleteConfirmed()
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.loading)
        assertEquals(false, viewModel.uiState.saving)
        assertEquals("", viewModel.uiState.nameInput)
        assertEquals("女", viewModel.uiState.sexInput)
        assertNull(viewModel.uiState.editingIdentityId)
        assertNull(viewModel.uiState.confirmDeleteIdentity)
        assertEquals("已删除 Alice", viewModel.uiState.message)
    }

    @Test
    fun `delete confirmed should clear dialog and surface repository error`() = runTest(mainDispatcherRule.dispatcher) {
        val target = sampleIdentity(id = "identity-1", name = "Alice")
        coEvery { repository.loadIdentities() } returns AppResult.Success(listOf(target))
        coEvery { repository.deleteIdentity(target.id) } returns AppResult.Error("删除失败")

        val viewModel = IdentityViewModel(repository)
        advanceUntilIdle()
        viewModel.confirmDelete(target)

        viewModel.deleteConfirmed()
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.saving)
        assertNull(viewModel.uiState.confirmDeleteIdentity)
        assertEquals("删除失败", viewModel.uiState.message)
    }

    private fun sampleIdentity(id: String, name: String): IdentityDto = IdentityDto(
        id = id,
        name = name,
        sex = "女",
        lastUsedAt = "2026-03-16 20:00:00",
    )
}
