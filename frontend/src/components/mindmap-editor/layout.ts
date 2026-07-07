import type { AINode } from './tree-utils';

export function calculateAutoLayout(nodes: AINode[]): AINode[] {
  // 1. Group by parent
  const childrenMap: Record<string, AINode[]> = {};
  let rootNode: AINode | null = null;

  nodes.forEach((n) => {
    if (n.parentId === null || n.parentId === '' || n.id === 'root') {
      rootNode = n;
    } else {
      if (!childrenMap[n.parentId]) {
        childrenMap[n.parentId] = [];
      }
      childrenMap[n.parentId].push(n);
    }
  });

  if (!rootNode) return nodes;

  const root = rootNode as AINode;

  // Sort child lists by order index
  Object.keys(childrenMap).forEach((key) => {
    childrenMap[key].sort((a, b) => a.order - b.order);
  });

  const nodePositions: Record<string, { x: number; y: number }> = {};
  const spacingX = 320;
  const spacingY = 140;

  // Recursive vertical size and placement calculation
  const layoutSubtree = (nodeId: string, currentX: number, currentY: number): number => {
    const children = childrenMap[nodeId] || [];
    nodePositions[nodeId] = { x: currentX, y: currentY };

    if (children.length === 0) {
      return spacingY; // leaf node space consumed
    }

    // Calculate total layout heights for all children subtrees
    const heights = children.map((c) => getSubtreeHeight(c.id, childrenMap));
    const totalHeight = heights.reduce((sum, h) => sum + h, 0);

    // Center children vertically with respect to parent node
    let nextY = currentY - totalHeight / 2 + heights[0] / 2;

    children.forEach((c, index) => {
      layoutSubtree(c.id, currentX + spacingX, nextY);
      if (index < children.length - 1) {
        nextY += (heights[index] + heights[index + 1]) / 2;
      }
    });

    return totalHeight;
  };

  // Helper to determine height of a sub-tree recursively
  const getSubtreeHeight = (nodeId: string, map: Record<string, AINode[]>): number => {
    const children = map[nodeId] || [];
    if (children.length === 0) {
      return spacingY;
    }
    return children.reduce((sum, c) => sum + getSubtreeHeight(c.id, map), 0);
  };

  layoutSubtree(root.id, 100, 300);

  return nodes.map((n) => ({
    ...n,
    position: n.position && n.position.x !== 0 ? n.position : (nodePositions[n.id] || { x: 0, y: 0 }),
  }));
}
