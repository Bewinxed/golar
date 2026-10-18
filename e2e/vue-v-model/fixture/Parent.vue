<script setup lang="ts">
	import { ref } from 'vue'
	import PropChild from './PropChild.vue'
	const str = ref('a')
</script>

<template>
	<PropChild v-model="str" :title="1" />
	<PropChild :model-value="1" v-model:title="str" />
	<PropChild :model-value="str" :title="1" />
</template>
