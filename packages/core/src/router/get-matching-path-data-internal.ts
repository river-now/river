import type { SemiDecoratedPath } from "./get-matching-path-data.js";

function get_matching_paths_internal(__paths: Array<SemiDecoratedPath>) {
  let paths = __paths.filter((x) => {
    // only continue if the path matches
    if (!x.matches) return false;

    // if it's dash route (home), no need to compare segments length
    if (x.realSegmentsLength === 0) return true;

    const index_adjusted_real_segments_length = x.isIndex
      ? x.realSegmentsLength + 1
      : x.realSegmentsLength;

    // make sure any remaining matches are not longer than the path itself
    return x.segments.length <= index_adjusted_real_segments_length;
  });

  // if there are multiple matches, filter out the ultimate catch-all
  if (paths.length > 1) {
    paths = paths.filter((x) => !x.isUltimateCatch);
  }

  let splat_segments: string[] = [];

  // if only one match now, return it
  if (paths.length === 1) {
    if (paths[0].isUltimateCatch) {
      splat_segments = paths[0].path.split("/").filter(Boolean);
    }

    return {
      splat_segments,
      paths,
    };
  }

  // now we only have real child paths

  // these are essentially any matching layout routes
  const definite_matches = paths.filter(
    (x) => !x.isIndex && !x.endsInDynamic && !x.endsInSplat,
  );

  const highest_scores_by_segment_length_of_definite_matches =
    definite_matches.reduce(
      (acc, x) => {
        const segment_length = x.segments.length;
        if (acc[segment_length] == null || x.score > acc[segment_length]) {
          acc[segment_length] = x.score;
        }
        return acc;
      },
      {} as Record<number, number>,
    );

  // the "maybe matches" need to compete with each other
  // they also need some more complicated logic
  const maybe_matches: SemiDecoratedPath[] = [];
  const grouped_by_segment_length: Record<number, SemiDecoratedPath[]> = {};

  for (const x of paths) {
    if (x.isIndex || x.endsInDynamic || x.endsInSplat) {
      const segment_length = x.segments.length;

      const highest_score_for_this_segment_length =
        highest_scores_by_segment_length_of_definite_matches[segment_length];

      if (
        highest_score_for_this_segment_length === undefined ||
        x.score > highest_score_for_this_segment_length
      ) {
        if (!grouped_by_segment_length[segment_length]) {
          grouped_by_segment_length[segment_length] = [];
        }
        grouped_by_segment_length[segment_length].push(x);
        maybe_matches.push(x);
      }
    }
  }

  const sorted_grouped_by_segment_length = Object.entries(
    grouped_by_segment_length,
  )
    .sort(([a], [b]) => Number(a) - Number(b))
    .map(([_, paths]) => paths);

  const xformed_maybes: SemiDecoratedPath[] = [];

  for (const paths of sorted_grouped_by_segment_length) {
    let winner = paths[0];
    let highest_score = winner.score;
    let index_candidate: SemiDecoratedPath | null = null;

    for (const path of paths) {
      if (path.isIndex && path.realSegmentsLength < path.segments.length) {
        index_candidate = path;
      }
      if (path.score > highest_score) {
        highest_score = path.score;
        winner = path;
      }
    }

    if (index_candidate) {
      winner = index_candidate;
    }

    const splat = paths.find((x) => x.endsInSplat);

    if (splat) {
      const data = winner.path.split("/").filter(Boolean);

      const number_of_non_splat_segments = winner.segments.filter(
        (x) => !x.isSplat,
      ).length;

      const number_of_splat_segments =
        data.length - number_of_non_splat_segments;

      splat_segments = data.slice(
        data.length - number_of_splat_segments,
        data.length,
      );
    }

    xformed_maybes.push(winner);
  }

  const maybe_final_paths = [...definite_matches, ...xformed_maybes].sort(
    (a, b) => a.segments.length - b.segments.length,
  );

  // if anything left
  if (maybe_final_paths.length) {
    const last_path = maybe_final_paths[maybe_final_paths.length - 1];

    // get index-adjusted segments length
    const last_path_segments_length_constructive = last_path.isIndex
      ? last_path.segments.length - 1
      : last_path.segments.length;

    const splat_too_far_out =
      last_path_segments_length_constructive > last_path.realSegmentsLength;

    const splat_needed =
      last_path_segments_length_constructive < last_path.realSegmentsLength;

    const not_a_splat = !last_path.endsInSplat;

    if (splat_too_far_out || (splat_needed && not_a_splat)) {
      return {
        splat_segments: paths[0].path.split("/").filter(Boolean),
        paths: __paths.filter((x) => x.matches && x.isUltimateCatch),
      };
    }
  }

  return {
    splat_segments,
    paths: maybe_final_paths,
  };
}

export { get_matching_paths_internal };
