package com.zcw.model;

/**
 * 删除操作结果
 */
public class DeleteResult {
    private int deletedRecords;    // 删除的数据库记录数
    private boolean fileDeleted;   // 文件是否已删除

    public DeleteResult() {
    }

    public DeleteResult(int deletedRecords, boolean fileDeleted) {
        this.deletedRecords = deletedRecords;
        this.fileDeleted = fileDeleted;
    }

    public int getDeletedRecords() {
        return deletedRecords;
    }

    public void setDeletedRecords(int deletedRecords) {
        this.deletedRecords = deletedRecords;
    }

    public boolean isFileDeleted() {
        return fileDeleted;
    }

    public void setFileDeleted(boolean fileDeleted) {
        this.fileDeleted = fileDeleted;
    }

    @Override
    public String toString() {
        return "DeleteResult{" +
                "deletedRecords=" + deletedRecords +
                ", fileDeleted=" + fileDeleted +
                '}';
    }
}
