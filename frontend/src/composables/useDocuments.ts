import { ref } from 'vue'
import type { Document } from '../types'
import * as api from '../api'

const documents = ref<Document[]>([])
const activeDocument = ref<Document | null>(null)
const loading = ref(false)

export function useDocuments() {
  async function loadDocuments() {
    loading.value = true
    try {
      documents.value = (await api.listDocuments()) || []
    } catch {
      documents.value = []
    } finally {
      loading.value = false
    }
  }

  async function selectDocument(id: number) {
    loading.value = true
    try {
      activeDocument.value = await api.getDocument(id)
    } catch {
      activeDocument.value = null
    } finally {
      loading.value = false
    }
  }

  function clearDocument() {
    activeDocument.value = null
  }

  async function removeDocument(id: number) {
    await api.deleteDocument(id)
    documents.value = documents.value.filter(d => d.id !== id)
    if (activeDocument.value?.id === id) {
      activeDocument.value = null
    }
  }

  return {
    documents,
    activeDocument,
    loading,
    loadDocuments,
    selectDocument,
    clearDocument,
    removeDocument,
  }
}
