import type { AINode, MindMapData } from './tree-utils';

export interface ValidationError {
  message: string;
}

export function detectCycle(nodes: AINode[]): boolean {
  for (const n of nodes) {
    const path = new Set<string>();
    let curr: string | null = n.id;
    while (curr && curr !== 'root') {
      if (path.has(curr)) {
        return true; // Cycle detected
      }
      path.add(curr);
      const parentNode = nodes.find((x) => x.id === curr);
      if (!parentNode) {
        break; // caught by parent existence checks
      }
      curr = parentNode.parentId;
    }
  }
  return false;
}

export function validateMindMapData(data: MindMapData): ValidationError | null {
  const nodes = data.nodes || [];
  const edges = data.edges || [];

  if (nodes.length === 0) {
    return { message: 'O mapa mental deve conter pelo menos um nó.' };
  }

  if (nodes.length > 150) {
    return { message: 'Limite excedido: o mapa não pode conter mais de 150 nós.' };
  }

  // 1. Root checks
  const roots = nodes.filter((n) => n.parentId === null || n.parentId === '' || n.id === 'root');
  if (roots.length !== 1) {
    return { message: `Deve existir exatamente um nó principal (root), foram encontrados ${roots.length}.` };
  }

  const rootNode = roots[0];
  if (rootNode.id !== 'root') {
    return { message: `O nó raiz principal deve ter o identificador 'root', obteve '${rootNode.id}'.` };
  }

  if (rootNode.parentId !== null && rootNode.parentId !== '') {
    return { message: 'O nó raiz principal (root) não pode possuir parentId.' };
  }

  // 2. Nodes constraints check
  const idSet = new Set<string>();
  for (const n of nodes) {
    if (!n.id) {
      return { message: 'Todos os nós devem conter um identificador (id) válido.' };
    }
    if (idSet.has(n.id)) {
      return { message: `Identificador de nó duplicado encontrado: ${n.id}` };
    }
    idSet.add(n.id);

    if (!n.title || n.title.trim() === '') {
      return { message: `O título do nó "${n.id}" não pode estar vazio.` };
    }

    if (n.title.length > 150) {
      return { message: `O título do nó "${n.title.substring(0, 15)}..." excede o limite de 150 caracteres.` };
    }

    if (n.content && n.content.length > 2000) {
      return { message: `O conteúdo do nó "${n.title}" excede o limite de 2000 caracteres.` };
    }

    // Non-root parent validation
    if (n.id !== 'root') {
      if (!n.parentId || n.parentId.trim() === '') {
        return { message: `O nó "${n.title}" deve possuir um parentId.` };
      }
      const parentExists = nodes.some((x) => x.id === n.parentId);
      if (!parentExists) {
        return { message: `O nó "${n.title}" aponta para um nó pai inexistente: "${n.parentId}".` };
      }
    }
  }

  // 3. Cycle detection
  if (detectCycle(nodes)) {
    return { message: 'Erro estrutural: o mapa mental contém ciclos direcionados.' };
  }

  // 4. Edges validation
  for (const e of edges) {
    const sourceExists = nodes.some((x) => x.id === e.source);
    const targetExists = nodes.some((x) => x.id === e.target);
    if (!sourceExists) {
      return { message: `A conexão aponta para uma origem inexistente: "${e.source}".` };
    }
    if (!targetExists) {
      return { message: `A conexão aponta para um destino inexistente: "${e.target}".` };
    }
  }

  return null;
}
