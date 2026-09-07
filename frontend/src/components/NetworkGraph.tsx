"use client";

import { useEffect, useRef } from "react";
import cytoscape from "cytoscape";
import CytoscapeComponent from "react-cytoscapejs";

interface NetworkGraphProps {
  elements: {
    nodes: any[];
    edges: any[];
  };
}

export default function NetworkGraph({ elements }: NetworkGraphProps) {
  const cyRef = useRef<cytoscape.Core | null>(null);

  useEffect(() => {
    if (cyRef.current) {
      // Re-layout when elements change
      cyRef.current.layout({ name: "concentric", fit: true, padding: 30 }).run();
    }
  }, [elements]);

  const stylesheet: cytoscape.Stylesheet[] = [
    {
      selector: "node",
      style: {
        label: "data(label)",
        color: "#fafafa",
        "text-valign": "bottom",
        "text-halign": "center",
        "text-margin-y": 6,
        "font-size": 10,
        "font-family": "Inter, sans-serif",
      },
    },
    {
      selector: 'node[type="entity"]',
      style: {
        "background-color": "#f59e0b", // Amber for entities
        shape: "hexagon",
        width: 40,
        height: 40,
        "border-width": 2,
        "border-color": "#b45309",
      },
    },
    {
      selector: 'node[type="address"]',
      style: {
        "background-color": "#3b82f6", // Blue for addresses
        shape: "ellipse",
        width: 30,
        height: 30,
      },
    },
    {
      selector: 'node[type="transaction"]',
      style: {
        "background-color": "#8b5cf6", // Purple for TX
        shape: "rectangle",
        width: 35,
        height: 25,
      },
    },
    {
      selector: 'node[type="utxo"]',
      style: {
        "background-color": "#10b981", // Green for UTXO
        shape: "round-rectangle",
        width: 20,
        height: 20,
      },
    },
    {
      selector: 'node[type="network_observation"]',
      style: {
        "background-color": "#ef4444", // Red for network obs
        shape: "diamond",
        width: 30,
        height: 30,
      },
    },
    {
      selector: "edge",
      style: {
        width: 1.5,
        "line-color": "#52525b", // zinc-600
        "target-arrow-color": "#52525b",
        "target-arrow-shape": "triangle",
        "curve-style": "bezier",
        label: "data(label)",
        "font-size": 8,
        "text-rotation": "autorotate",
        "text-margin-y": -5,
        color: "#a1a1aa",
      },
    },
  ];

  if (!elements || !elements.nodes) {
    return <div className="w-full h-full flex items-center justify-center text-muted-foreground">No graph data available.</div>;
  }

  return (
    <div className="w-full h-full bg-black/20 rounded-lg overflow-hidden border border-border">
      <CytoscapeComponent
        elements={CytoscapeComponent.normalizeElements(elements)}
        stylesheet={stylesheet}
        style={{ width: "100%", height: "100%" }}
        layout={{ name: "concentric" }}
        cy={(cy) => {
          cyRef.current = cy;
          // Setup tooltips or interactions if needed
          cy.on('tap', 'node', function(evt){
            var node = evt.target;
            console.log('Tapped ' + node.id());
          });
        }}
      />
    </div>
  );
}
