<script setup lang="ts">
	defineProps<{ modelValue: number; title: number }>()
	defineEmits<{
		'update:modelValue': [v: number]
		'update:title': [v: number]
	}>()
</script>

<template><div /></template>
