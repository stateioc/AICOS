<template>
  <div class="example1-wrapper">
    <div class="fr clearfix mb15">
      <bk-form form-type="inline">
        <bk-form-item label="名称">
          <bk-input placeholder="名称" v-model="formData.name" />
        </bk-form-item>
        <bk-form-item label="日期">
          <bk-date-picker placeholder="日期" :timer="false" v-model="formData.date" />
        </bk-form-item>
        <bk-form-item>
          <bk-button theme="primary" title="提交">搜索</bk-button>
        </bk-form-item>
      </bk-form>
    </div>

    <bk-table style="margin-top: 15px;"
              :data="tableData"
              :size="size"
              :pagination="pagination"
              @page-change="handlePageChange">
      <bk-table-column type="selection" width="60" align="center" />
      <bk-table-column type="index" label="序列" align="center" width="60" />
      <bk-table-column label="名称/内网IP" prop="ip" />
      <bk-table-column label="来源" prop="source" />
      <bk-table-column label="状态" prop="status" />
      <bk-table-column label="创建时间" prop="create_time" />
      <bk-table-column label="操作" width="150">
        <template slot-scope="props">
          <bk-button theme="primary"
                     text :disabled="props.row.status === '创建中'" @click="reset(props.row)">重置</bk-button>
          <bk-button theme="primary" text @click="remove(props.row)">移除</bk-button>
        </template>
      </bk-table-column>
    </bk-table>
  </div>
</template>

<script>
export default {
  components: {
  },
  data() {
    return {
      size: 'small',
      formData: {
        name: '',
        date: '',
      },
      tableData: [],
      pagination: {
        current: 1,
        count: 0,
        limit: 10,
      },
    };
  },
  created() {
    this.init();
  },
  methods: {
    init() {
      this.getTableData();
    },
    async getTableData() {
      try {
        const res = await this.$store.dispatch('example/getTableData', {}, { fromCache: true });
        this.tableData = res.data.list;
        this.pagination.count = res.data.list.length;
      } catch (e) {
        console.error(e);
      }
    },
    toggleTableSize() {
      const size = ['small', 'medium', 'large'];
      const index = (size.indexOf(this.size) + 1) % 3;
      this.size = size[index];
    },
    handlePageChange(page) {
      this.pagination.current = page;
    },
    remove(row) {
      const index = this.tableData.indexOf(row);
      if (index !== -1) {
        this.tableData.splice(index, 1);
      }
    },
    reset(row) {
      // eslint-disable-next-line no-param-reassign
      row.status = '创建中';
    },
  },
};
</script>
