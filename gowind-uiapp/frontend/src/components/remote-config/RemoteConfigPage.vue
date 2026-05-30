<script setup lang="ts">

</script>

<template>
  <div class="code-generator-container">
    <a-card title="远程配置" class="full-card">
      <!-- Card 内容 -->
    </a-card>
  </div>
</template>

<style scoped>
.code-generator-container {
  width: 100%;
  height: 100%;
  padding: 0;
  margin: 0;
  box-sizing: border-box;
}

.full-card {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
}

:deep(.ant-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

:deep(.ant-card-head) {
  flex-shrink: 0;
}

:deep(.ant-card-body) {
  flex: 1;
  overflow: auto;
  padding: 16px;
  box-sizing: border-box;
}
</style>
