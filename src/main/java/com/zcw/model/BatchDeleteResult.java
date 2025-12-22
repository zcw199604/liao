package com.zcw.model;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * 批量删除操作结果
 */
public class BatchDeleteResult {
    private int successCount;                       // 成功删除的数量
    private int failCount;                          // 失败的数量
    private List<Map<String, String>> failedItems;  // 失败项列表

    public BatchDeleteResult() {
        this.failedItems = new ArrayList<>();
    }

    public BatchDeleteResult(int successCount, int failCount, List<Map<String, String>> failedItems) {
        this.successCount = successCount;
        this.failCount = failCount;
        this.failedItems = failedItems != null ? failedItems : new ArrayList<>();
    }

    public int getSuccessCount() {
        return successCount;
    }

    public void setSuccessCount(int successCount) {
        this.successCount = successCount;
    }

    public int getFailCount() {
        return failCount;
    }

    public void setFailCount(int failCount) {
        this.failCount = failCount;
    }

    public List<Map<String, String>> getFailedItems() {
        return failedItems;
    }

    public void setFailedItems(List<Map<String, String>> failedItems) {
        this.failedItems = failedItems;
    }

    @Override
    public String toString() {
        return "BatchDeleteResult{" +
                "successCount=" + successCount +
                ", failCount=" + failCount +
                ", failedItems=" + failedItems +
                '}';
    }
}
